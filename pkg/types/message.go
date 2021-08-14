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

type MessageContentInfo struct {
	BlobUUID uuid.UUID `json:"blobUUID,omitempty"`
	Filename string    `json:"filename,omitempty"`
	Mimetype string    `json:"mimetype,omitempty"`
}

type MessageMeta struct {
	Sender      string             `json:"sender"`
	Time        time.Time          `json:"time"`
	Type        MessageType        `json:"type"`
	ContentInfo MessageContentInfo `json:"contentMeta,omitempty"`
}

type Message struct {
	Meta    MessageMeta `json:"meta"`
	Content []byte      `json:"content,omitempty"`
}
