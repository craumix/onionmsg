package types

import (
	"crypto/ed25519"
	"time"
)

type Message struct {
	Sender		string		`json:"sender"`
	Time		time.Time	`json:"time"`
	Content		[]byte		`json:"content"`
	Signature	[]byte		`json:"signature"`
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
	d = append(d, m.Content...)

	return d
}