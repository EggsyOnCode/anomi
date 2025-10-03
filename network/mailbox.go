package network

import "sync"

type Mailbox struct {
	outCh     chan []byte
	mu        *sync.RWMutex
	transport *LibP2pTransport
}

// TODO: also need info about node and api server in args to construct the RPC.msg && send inbox to the api server
func NewMailbox(p2p *LibP2pTransport) *Mailbox {
	out := make(chan []byte, 1000)

	box := &Mailbox{
		outCh:     out,
		transport: p2p,
	}

	go box.broadcast()
	go box.listen()

	return box
}

// called when the caller wishes to send a msg to the p2p network
// use the msg as payload to constrcut RPC.msg
func (m *Mailbox) Out(msg []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.outCh <- msg
}

func (m *Mailbox) broadcast() {
	for msg := range m.outCh {
		m.transport.Broadcast(msg)
	}
}

func (m *Mailbox) listen() {
	msgs := m.transport.ConsumeMsgs()
	for msg := range msgs {
		// TODO: replace this with api gateway or something else that should be handling the msg
		m.outCh <- msg.Payload
	}
}

func (m *Mailbox) Stop() {
	close(m.outCh)
}
