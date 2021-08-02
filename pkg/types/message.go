package types

import (
	"time"
)

type MessageType string

const (
	MessageTypeText MessageType = "mtype.text"
	MessageTypeCmd MessageType = "mtype.cmd"
	MessageTypeBlob MessageType = "mtype.blob"
)

type MessageMeta struct {
	Sender string      `json:"sender"`
	Time   time.Time   `json:"time"`
	Type   MessageType `json:"type"`
}

type Message struct {
	Meta    MessageMeta `json:"meta"`
	Content []byte      `json:"content"`
}
