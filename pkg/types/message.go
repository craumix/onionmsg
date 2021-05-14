package types

import (
	"time"
)

const (
	MTYPE_TEXT = 0x00
	MTYPE_CMD  = 0x01
	MTYPE_BLOB = 0x02
)

type MessageMeta struct {
	Sender string    `json:"sender"`
	Time   time.Time `json:"time"`
	Type   byte      `json:"type"`
}

type Message struct {
	Meta    MessageMeta `json:"meta"`
	Content []byte      `json:"content"`
}
