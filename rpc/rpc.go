package rpc

import (
	"bytes"
	"encoding/gob"

	"github.com/EggysOnCode/anomi/crypto"
)

type RPCMessage struct {
	FromID    string           `json:"from_id"`
	Payload   []byte           `json:"payload"`
	Signature crypto.Signature `json:"signature"`
}

// payload here will be of type Message but in serialized bytes format
func NewRPCMessage(from string, payload []byte, fromID string) *RPCMessage {
	return &RPCMessage{
		FromID:  fromID,
		Payload: payload,
	}
}

func (m *RPCMessage) Bytes(codec Codec) ([]byte, error) {
	return codec.Encode(m)
}

// after receiving a msg, it must first be decoded by
// Codec, then passed to RPCProcessor
type DecodedMsg struct {
	FromId    string
	Data      *InternalMessage // Use InternalMessage instead of generic any
	Signature crypto.Signature
}


type RPCDecodeFunc func(RPCMessage, Codec) (*DecodedMsg, error)

type InternalPeerServerInfoMsg struct {
	NetworkId  string
	ListenAddr string
	ServerId   string
}

func (m *InternalPeerServerInfoMsg) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(m); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
