package types

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/json"
	"log"
	"time"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/google/uuid"
)

const (
	MTYPE_TEXT = 0x00
	MTYPE_CMD  = 0x01
	MTYPE_BLOB = 0x02
)

/*
Message is a struct that contains the data and metadata for a message.
The RawContent is usually not the actual content, because for Blobs the Blobmanager is used
to store them on the disk, in which case only a uuid is saved
*/
type Message struct {
	Sender     string    `json:"sender"`
	Time       time.Time `json:"time"`
	Type       byte      `json:"type"`
	RawContent []byte    `json:"content"`
	Signature  []byte    `json:"signature"`
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
		Sender:     fingerprint,
		Time:       time.Now(),
		Type:       mtype,
		RawContent: content,
	}, nil
}

/*
Sign creates a signature from the Sender, Timestamp, Type and Content using the provided PrivateKey.
*/
func (m *Message) Sign(priv ed25519.PrivateKey) {
	m.Signature = ed25519.Sign(priv, m.digestBytes())
}

/*
Verify verifies the signature of this Message against the provided PublicKey.
*/
func (m *Message) Verify(pub ed25519.PublicKey) bool {
	if m.Signature == nil {
		return false
	}

	return ed25519.Verify(pub, m.digestBytes(), m.Signature)
}

func (m *Message) digestBytes() []byte {
	time := make([]byte, 8)
	binary.LittleEndian.PutUint64(time, uint64(m.Time.Unix()))

	d := []byte(m.Sender)
	d = append(d, time...)
	d = append(d, m.Type)
	d = append(d, m.GetContent()...)

	return d
}

/*
GetContent returns the real Content from this Message, as opposed to only a uuid if the type is MTYPE_BLOB.
*/
func (m *Message) GetContent() (res []byte) {
	if m.Type == MTYPE_BLOB {
		blobid, err := uuid.ParseBytes(m.RawContent)
		if err != nil {
			log.Println(err.Error())
			return
		}
		res, err = blobmngr.GetRessource(blobid)
		if err != nil {
			log.Println(err.Error())
			return
		}

		return res
	}

	return m.RawContent
}

/*
AsRealContentJSON returns a marshaled Message struct with the RawContent replaced by the Content from getContent().
*/
func (m *Message) AsRealContentJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sender    string    `json:"sender"`
		Time      time.Time `json:"time"`
		Type      byte      `json:"type"`
		Content   []byte    `json:"content"`
		Signature []byte    `json:"signature"`
	}{
		Sender:    m.Sender,
		Time:      m.Time,
		Type:      m.Type,
		Content:   m.GetContent(),
		Signature: m.Signature,
	})
}

/*
MessageFromRealContentJSON unmarshals a marshaled Message from AsRealContentJSON().
*/
func MessageFromRealContentJSON(b []byte) (*Message, error) {
	msg := &Message{}
	err := json.Unmarshal(b, msg)
	if err != nil {
		return nil, err
	}

	content, err := parseNewContent(msg.RawContent, msg.Type)
	if err != nil {
		return nil, err
	}

	msg.RawContent = content

	return msg, nil
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
