package types

import (
	"crypto/ed25519"
	"encoding/base64"
)

type Identity struct {
	Key   ed25519.PrivateKey `json:"key"`
	Nick  string             `json:"nick"`
	Admin bool               `json:"admin"`
}

func NewIdentity() Identity {
	_, priv, _ := ed25519.GenerateKey(nil)

	return Identity{
		Key: priv,
	}
}

func (i *Identity) Fingerprint() string {
	return i.B64PubKey()
}

func (i *Identity) B64PubKey() string {
	return base64.RawURLEncoding.EncodeToString(i.Key.Public().(ed25519.PublicKey))
}

func (i *Identity) Sign(data []byte) []byte {
	return ed25519.Sign(i.Key, data)
}
