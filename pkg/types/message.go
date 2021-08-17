package types

import (
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypeText    MessageType = "mtype.text"
	MessageTypeCmd     MessageType = "mtype.cmd"
	MessageTypeBlob    MessageType = "mtype.blob"
	MessageTypeSticker MessageType = "mtype.sticker"
)

type ContentMeta struct {
	BlobUUID uuid.UUID `json:"blobUUID,omitempty"`
	Filename string    `json:"filename,omitempty"`
	Mimetype string    `json:"mimetype,omitempty"`
}

type MessageMeta struct {
	Sender string    `json:"sender"`
	Time   time.Time `json:"time"`
}

type MessageContent struct {
	Type MessageType `json:"type"`
	Meta ContentMeta `json:"meta,omitempty"`
	Data []byte      `json:"data,omitempty"`
}

type Message struct {
	Meta    MessageMeta    `json:"meta"`
	Content MessageContent `json:"cotent,omitempty"`
}
