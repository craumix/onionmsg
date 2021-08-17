package types

import (
	"time"

	"github.com/google/uuid"
)

type ContentType string

const (
	MessageTypeText    ContentType = "mtype.text"
	MessageTypeCmd     ContentType = "mtype.cmd"
	MessageTypeFile    ContentType = "mtype.file"
	MessageTypeSticker ContentType = "mtype.sticker"
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
	Type ContentType `json:"type"`
	Meta ContentMeta `json:"meta,omitempty"`
	Data []byte      `json:"data,omitempty"`
}

type Message struct {
	Meta    MessageMeta    `json:"meta"`
	Content MessageContent `json:"cotent,omitempty"`
}

func (m *Message) ContainsBlob() bool {
	return m.Content.Meta.BlobUUID != uuid.Nil
}
