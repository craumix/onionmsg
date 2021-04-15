package types

import (
	"crypto/ed25519"
	"time"
)

type Message struct {
	Sender		string
	Time		time.Time
	Content		[]byte
	signature	[]byte
}

func (m *Message) Sign(priv ed25519.PrivateKey) {
	
	m.signature = ed25519.Sign(priv, m.digestBytes())
}

func (m *Message) Verify(pub ed25519.PublicKey) bool {
	if m.signature == nil {
		return false
	}

	return ed25519.Verify(pub, m.digestBytes(), m.signature)
}

func (m *Message) digestBytes() []byte {
	d := []byte(m.Sender)
	d = append(d, int64ToBytes(m.Time.Unix())...)
	d = append(d, m.Content...)

	return d
}