package types

import (
	"crypto/ed25519"
)

type IdentityMeta struct {
	Nick  string `json:"nick"`
	Admin bool   `json:"admin"`
}

type Identity struct {
	Key  ed25519.PrivateKey `json:"key"`
	Meta IdentityMeta       `json:"meta"`
}

func NewIdentity() Identity {
	_, priv, _ := ed25519.GenerateKey(nil)

	return Identity{
		Key: priv,
	}
}

func (i *Identity) Fingerprint() string {
	return Fingerprint(i.Key.Public().(ed25519.PublicKey))
}

func (i *Identity) Sign(data []byte) []byte {
	return Sign(i.Key, data)
}
