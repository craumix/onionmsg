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

type Message struct {
	Sender     string    `json:"sender"`
	Time       time.Time `json:"time"`
	Type       byte      `json:"type"`
	RawContent []byte    `json:"content"`
	Signature  []byte    `json:"signature"`
}

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

func (m *Message) Sign(priv ed25519.PrivateKey) {
	m.Signature = ed25519.Sign(priv, m.digestBytes())
}

func (m *Message) Verify(pub ed25519.PublicKey) bool {
	if m.Signature == nil {
		return false
	}

	return ed25519.Verify(pub, m.digestBytes(), m.Signature)
}

func (m *Message) digestBytes() []byte {
	d := []byte(m.Sender)
	d = append(d, int64ToBytes(m.Time.Unix())...)
	d = append(d, m.Type)
	d = append(d, m.GetContent()...)

	return d
}

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

func (m *Message) MarshalJSON() ([]byte, error) {
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

func (m *Message) UnmarshalJSON(b []byte) error {
	i := struct {
		Sender    string    `json:"sender"`
		Time      time.Time `json:"time"`
		Type      byte      `json:"type"`
		Content   []byte    `json:"content"`
		Signature []byte    `json:"signature"`
	}{}
	err := json.Unmarshal(b, &i)
	if err != nil {
		return err
	}

	content, err := parseNewContent(i.Content, i.Type)
	if err != nil {
		return err
	}

	*m = Message{
		Sender:     i.Sender,
		Time:       i.Time,
		Type:       i.Type,
		RawContent: content,
		Signature:  i.Signature,
	}

	return nil
}

func int64ToBytes(i int64) []byte {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, uint64(i))
	return bs
}

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
