package types

import (
	"crypto/ed25519"
	"encoding/base64"
)

type Identity struct {
	Service *HiddenService
	Pub 	ed25519.PublicKey
	Priv	ed25519.PrivateKey
}

func NewIdentity() *Identity {
	pub, priv, _ := ed25519.GenerateKey(nil)

	return &Identity {
		Service: NewHiddenService(),
		Pub: pub,
		Priv: priv,
	}
}

func (i *Identity) Fingerprint() string {
	return i.B64PubKey() + "@" + i.Service.Onion().ServiceID
}

func (i *Identity) B64PubKey() string {
	return base64.RawURLEncoding.EncodeToString(i.Pub)
}