package types

import (
	"time"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/google/uuid"
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

/*
NewMessage creates a new Message struct and if the Message if of type MTYPE_BLOB,
saves the Blob to disk using the Blobmanager.
*/
func NewMessage(fingerprint string, mtype byte, raw []byte) (*Message, error) {
	content, err := parseNewContent(raw, mtype)
	if err != nil {
		return nil, err
	}

	return &Message{
		Meta: MessageMeta{
			Sender: fingerprint,
			Time:   time.Now(),
			Type:   mtype,
		},
		Content: content,
	}, nil
}

/*
Returns the a uuid for the Blobmanager if the type is MTYPE_BLOB, else returns the original bytes.
*/
func parseNewContent(raw []byte, mtype byte) (content []byte, err error) {
	if mtype == MTYPE_BLOB {
		var id uuid.UUID
		id, err = blobmngr.SaveRessource(raw)
		if err != nil {
			return
		}
		content = id[:]
	} else {
		content = raw
	}

	return
}
