package types

import (
	"time"

	"github.com/google/uuid"
)

type ContentType string

const (
	ContentTypeText    ContentType = "mtype.text"
	ContentTypeCmd     ContentType = "mtype.cmd"
	ContentTypeFile    ContentType = "mtype.file"
	ContentTypeSticker ContentType = "mtype.sticker"
)

type ContentMeta struct {
	BlobUUID uuid.UUID `json:"blobUUID"`
	Filename string    `json:"filename,omitempty"`
	Mimetype string    `json:"mimetype,omitempty"`
}

type MessageMeta struct {
	Sender string    `json:"sender"`
	Time   time.Time `json:"time"`
}

type MessageContent struct {
	Type ContentType `json:"type"`
	Meta ContentMeta `json:"meta"`
	Data []byte      `json:"data,omitempty"`
}

type Message struct {
	Meta    MessageMeta    `json:"meta"`
	Content MessageContent `json:"cotent"`
}

func (m *Message) ContainsBlob() bool {
	return m.Content.Meta.BlobUUID != uuid.Nil
}
