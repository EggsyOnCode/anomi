package network

import (
	"fmt"
	"sync"
	"time"

	"github.com/EggysOnCode/anomi/api/controllers/p2p"
	"github.com/EggysOnCode/anomi/crypto"
	"github.com/EggysOnCode/anomi/logger"
	"github.com/EggysOnCode/anomi/rpc"
	"go.uber.org/zap"
)

// ProtocolPeer represents a peer in our protocol
type ProtocolPeer struct {
	ServerID string // server ID (application level ID)
	*Peer
}

// ServerOpts contains configuration for the P2P server
type ServerOpts struct {
	ListenAddr     string
	CodecType      rpc.CodecType
	BootstrapNodes []string           // Network addresses of bootstrap nodes
	PrivateKey     *crypto.PrivateKey // Node's private key for identity
	id             crypto.PublicKey   // derived from PrivateKey
}

// Server handles P2P network operations
type Server struct {
	*ServerOpts
	Transporter   Transport
	RPCProcessor  *ServerRPCProcessor
	peerLock      *sync.RWMutex // protects peer operations
	Codec         rpc.Codec
	quitCh        chan struct{} // channel for signals to stop server
	peers         map[string]*ProtocolPeer
	appIdToPeerId map[string]string // maps application ID to network ID
	p2pController *p2p.P2PController
	peerCountChan chan int
	logger        *zap.SugaredLogger
}

// ID returns the server's public key
func (s *Server) ID() crypto.PublicKey {
	return s.id
}

// NewServer creates a new P2P server
func NewServer(opts *ServerOpts, controller *p2p.P2PController) *Server {
	// Create message channel for transport layer
	msgCh := make(chan *rpc.RPCMessage, 1024)

	// Create transport layer
	transporter := NewLibp2pTransport(msgCh, &rpc.JSONCodec{}, *opts.PrivateKey)

	// Create RPC processor
	rpcProcessor := &ServerRPCProcessor{
		logger:            logger.Get().Sugar(),
		controller:        controller,
		commsChWithServer: make(chan *rpc.InternalPeerServerInfoMsg, 1000),
	}

	s := &Server{
		Transporter:   transporter,
		p2pController: controller,
		RPCProcessor:  rpcProcessor,
		peerLock:      &sync.RWMutex{},
		ServerOpts:    opts,
		quitCh:        make(chan struct{}),
		appIdToPeerId: make(map[string]string),
		peers:         make(map[string]*ProtocolPeer),
		peerCountChan: make(chan int, 1000),
	}

	// Set server's ID from private key
	s.id = s.PrivateKey.PublicKey()

	// Set up codec
	switch s.CodecType {
	case rpc.JsonCodec:
		s.Codec = rpc.NewJsonCodec()
	default:
		panic(fmt.Errorf("unsupported codec type: %v", s.CodecType))
	}

	s.logger = logger.Get().Sugar().With("server_id", s.id.String())

	// Set codec in transport
	transporter.SetCodec(s.Codec)

	return s
}

// Start starts the P2P server
func (s *Server) Start() {
	// Start transport layer
	go s.Transporter.Start()
	defer s.Transporter.Stop()

	// Wait for transport to initialize
	time.Sleep(2 * time.Second)

	// Handle new peer connections
	go func() {
		for peer := range s.Transporter.ConsumePeers() {
			s.handleNewPeer(peer)
		}
	}()

	// Handle peer info messages
	go func() {
		for infoMsg := range s.RPCProcessor.commsChWithServer {
			s.peerLock.Lock()

			peer, ok := s.peers[infoMsg.NetworkId]
			if !ok {
				s.logger.Error("peer not found in peer map")
				s.peerLock.Unlock()
				continue
			}

			// Map application ID to network ID
			s.appIdToPeerId[infoMsg.ServerId] = infoMsg.NetworkId
			peer.ServerID = infoMsg.ServerId

			s.peerLock.Unlock()
		}
	}()

	// Main message processing loop
free:
	for {
		select {
		case rpcMsg := <-s.Transporter.ConsumeMsgs():
			// Decode RPC message
			decodedMsg, err := s.RPCProcessor.DefaultRPCDecoder(rpcMsg, s.Codec)
			if err != nil {
				s.logger.Errorf("error decoding RPC message: %v", err)
				continue
			}
			
			// handle in case of handshake
			if err := s.RPCProcessor.HandleHandshake(decodedMsg); err != nil {
				s.logger.Errorf("error processing RPC message: %v", err)
			}

			// otherwise send to controller for processing
			if _, err := s.p2pController.HandleMessage(rpcMsg); err != nil {
				s.logger.Errorf("error processing RPC message: %v", err)
			}

		case <-s.quitCh:
			break free
		}
	}
	s.logger.Info("P2P server stopped")
}

// sendHandshakeMsgToPeerNode sends a handshake message to a peer
func (s *Server) sendHandshakeMsgToPeerNode(id string) error {
	// Create handshake message
	handshakeMsg := &rpc.InternalPeerServerInfoMsg{
		NetworkId:  s.Transporter.ID(),
		ListenAddr: string(s.Transporter.Addr()),
		ServerId:   s.ID().String(),
	}

	// Create internal message
	internalMsg, err := rpc.NewHandshakeMessage(handshakeMsg)
	if err != nil {
		return fmt.Errorf("failed to create internal message: %w", err)
	}

	// Create RPC message
	rpcMsg, err := internalMsg.CreateRPCMessage(string(s.Transporter.Addr()), s.ID().String(), s.Codec)
	if err != nil {
		return fmt.Errorf("failed to create RPC message: %w", err)
	}

	// Serialize RPC message
	msgBytes, err := rpcMsg.Bytes(s.Codec)
	if err != nil {
		return fmt.Errorf("failed to serialize RPC message: %w", err)
	}

	s.logger.Debug("sending handshake with payload length: ", len(msgBytes))

	return s.Transporter.SendMsg(id, msgBytes)
}

// SendMsg sends a message to a specific peer by application ID
func (s *Server) SendMsg(appID string, internalMsg *rpc.InternalMessage) error {
	// Get corresponding network ID
	s.peerLock.RLock()
	networkID := s.appIdToPeerId[appID]
	s.peerLock.RUnlock()

	if networkID == "" {
		return fmt.Errorf("peer not found for app ID: %s", appID)
	}

	// Create RPC message
	rpcMsg, err := internalMsg.CreateRPCMessage(string(s.Transporter.Addr()), s.ID().String(), s.Codec)
	if err != nil {
		return fmt.Errorf("failed to create RPC message: %w", err)
	}

	// Serialize RPC message
	msgBytes, err := rpcMsg.Bytes(s.Codec)
	if err != nil {
		return fmt.Errorf("failed to serialize RPC message: %w", err)
	}

	return s.Transporter.SendMsg(networkID, msgBytes)
}

// BroadcastMsg broadcasts a message to all connected peers
func (s *Server) BroadcastMsg(internalMsg *rpc.InternalMessage) error {
	// Create RPC message
	rpcMsg, err := internalMsg.CreateRPCMessage(string(s.Transporter.Addr()), s.ID().String(), s.Codec)
	if err != nil {
		return fmt.Errorf("failed to create RPC message: %w", err)
	}

	// Serialize RPC message
	msgBytes, err := rpcMsg.Bytes(s.Codec)
	if err != nil {
		return fmt.Errorf("failed to serialize RPC message: %w", err)
	}

	s.Transporter.Broadcast(msgBytes)
	return nil
}

// Stop stops the P2P server
func (s *Server) Stop() {
	close(s.quitCh)
}

// GetCodec returns the server's codec
func (s *Server) GetCodec() rpc.Codec {
	return s.Codec
}

// GetPeerCount returns the number of connected peers
func (s *Server) GetPeerCount() int {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()
	return len(s.peers)
}

// IsPeer checks if a peer is connected by application ID
func (s *Server) IsPeer(appID string) bool {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()

	_, ok := s.appIdToPeerId[appID]
	return ok
}

// GetPeerList returns a list of all connected peer application IDs
func (s *Server) GetPeerList() []string {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()

	var peerList []string
	for appID := range s.appIdToPeerId {
		peerList = append(peerList, appID)
	}
	return peerList
}

// handleNewPeer handles a new peer connection
func (s *Server) handleNewPeer(peer *Peer) {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	// Create protocol peer
	pPeer := &ProtocolPeer{
		ServerID: "",
		Peer:     peer,
	}

	// Add to peer map
	s.peers[peer.ID] = pPeer
	s.peerCountChan <- len(s.peers)

	// Send handshake message
	if err := s.sendHandshakeMsgToPeerNode(peer.ID); err != nil {
		s.logger.Errorf("failed to send handshake to peer %s: %v", peer.ID, err)
	}
}

// ConnectToPeer connects to a peer by network address
func (s *Server) ConnectToPeer(addr string) error {
	// The transport layer handles peer discovery
	s.Transporter.DiscoverPeers()
	return nil
}

// GetPeerCountChannel returns the channel for peer count updates
func (s *Server) GetPeerCountChannel() <-chan int {
	return s.peerCountChan
}
