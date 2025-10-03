package rpc

import (
	"fmt"
)

type MessageType byte
type MesageTopic byte

const (
	DB MesageTopic = iota
)

type Message struct {
	Headers MessageType // this is for Routing / switching on MsgType
	Topic   MesageTopic
	Data    []byte
}

func NewMessage(headers MessageType, data []byte) *Message {
	return &Message{
		Headers: headers,
		Data:    data,
	}
}

func (m *Message) Bytes(c Codec) ([]byte, error) {
	return c.Encode(m)
}

func (m *Message) String() string {
	return fmt.Sprintf("topic %v, headers %v , data %s ", m.Topic, m.Headers, m.Data)
}
