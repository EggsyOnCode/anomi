package rpc

import (
	"fmt"

	"github.com/EggysOnCode/anomi/crypto"
)

// RPCMessageBuilder builds RPC messages from internal messages
type RPCMessageBuilder struct {
	fromSock    string
	fromID      string
	codec       Codec
	internalMsg *InternalMessage
	signature   crypto.Signature
}

// NewRPCMessageBuilder creates a new RPC message builder
func NewRPCMessageBuilder(fromSock, fromID string, codec Codec) *RPCMessageBuilder {
	return &RPCMessageBuilder{
		fromSock: fromSock,
		fromID:   fromID,
		codec:    codec,
	}
}

// SetInternalMessage sets the internal message to be packaged
func (b *RPCMessageBuilder) SetInternalMessage(msg *InternalMessage) *RPCMessageBuilder {
	b.internalMsg = msg
	return b
}

// SetSignature sets the signature for the RPC message
func (b *RPCMessageBuilder) SetSignature(sig crypto.Signature) *RPCMessageBuilder {
	b.signature = sig
	return b
}

// Build constructs the RPCMessage from the internal message
func (b *RPCMessageBuilder) Build() (*RPCMessage, error) {
	if b.internalMsg == nil {
		return nil, fmt.Errorf("internal message cannot be nil")
	}

	// Serialize the internal message to bytes
	payload, err := b.internalMsg.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize internal message: %w", err)
	}

	// Create the RPC message
	return &RPCMessage{
		FromID:    b.fromID,
		Payload:   payload,
		Signature: b.signature,
	}, nil
}

// Bytes serializes the RPC message to bytes using the codec
func (b *RPCMessageBuilder) Bytes() ([]byte, error) {
	rpcMsg, err := b.Build()
	if err != nil {
		return nil, err
	}
	return rpcMsg.Bytes(b.codec)
}

// BuildFromInternalMessage is a convenience function to build RPC message directly from internal message
func BuildFromInternalMessage(fromSock, fromID string, codec Codec, internalMsg *InternalMessage) (*RPCMessage, error) {
	builder := NewRPCMessageBuilder(fromSock, fromID, codec)
	return builder.SetInternalMessage(internalMsg).Build()
}

// BuildFromInternalMessageWithSignature is a convenience function to build RPC message with signature
func BuildFromInternalMessageWithSignature(fromSock, fromID string, codec Codec, internalMsg *InternalMessage, sig crypto.Signature) (*RPCMessage, error) {
	builder := NewRPCMessageBuilder(fromSock, fromID, codec)
	return builder.SetInternalMessage(internalMsg).SetSignature(sig).Build()
}
