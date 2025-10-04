package network

import (
	"fmt"

	"github.com/EggysOnCode/anomi/api/controllers/p2p"
	"github.com/EggysOnCode/anomi/rpc"
	"go.uber.org/zap"
)

// ServerRPCProcessor handles RPC message processing and routing
type ServerRPCProcessor struct {
	logger            *zap.SugaredLogger
	controller *p2p.P2PController
	commsChWithServer chan *rpc.InternalPeerServerInfoMsg
}

// NewServerRPCProcessor creates a new RPC processor
func NewServerRPCProcessor() *ServerRPCProcessor {
	return &ServerRPCProcessor{
		logger:            nil, // Will be set by server
		commsChWithServer: make(chan *rpc.InternalPeerServerInfoMsg, 1000),
	}
}

// ProcessMessage processes a decoded RPC message
func (s *ServerRPCProcessor) HandleHandshake(msg *rpc.DecodedMsg) error {
	if msg.Data == nil {
		return fmt.Errorf("message data is nil")
	}

	// Get the internal message
	internalMsg := msg.Data

	// Handle handshake messages directly
	if internalMsg.Type == rpc.Handshake {
		// This is a handshake message
		var handshakeMsg rpc.InternalPeerServerInfoMsg
		if err := internalMsg.UnmarshalData(&handshakeMsg); err != nil {
			return fmt.Errorf("failed to unmarshal handshake message: %w", err)
		}

		// Send to server communication channel
		select {
		case s.commsChWithServer <- &handshakeMsg:
		default:
			s.logger.Warn("comms channel is full, dropping handshake message")
		}

		return nil
	}

	return nil
}

// DefaultRPCDecoder decodes an RPC message into a DecodedMsg
func (s *ServerRPCProcessor) DefaultRPCDecoder(rpcMsg *rpc.RPCMessage, codec rpc.Codec) (*rpc.DecodedMsg, error) {
	// Decode the RPC message payload into an internal message
	internalMsg, err := rpc.FromBytes(rpcMsg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode internal message: %w", err)
	}

	// Create decoded message
	decodedMsg := &rpc.DecodedMsg{
		FromId:    rpcMsg.FromID,
		Data:      internalMsg,
		Signature: rpcMsg.Signature,
	}

	return decodedMsg, nil
}

// GetCommsChannel returns the communication channel for handshake messages
func (s *ServerRPCProcessor) GetCommsChannel() <-chan *rpc.InternalPeerServerInfoMsg {
	return s.commsChWithServer
}

// SetLogger sets the logger for the RPC processor
func (s *ServerRPCProcessor) SetLogger(logger *zap.SugaredLogger) {
	s.logger = logger
}
