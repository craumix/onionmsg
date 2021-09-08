package types

import (
	"crypto/ed25519"
)

type ContactIdentity ed25519.PrivateKey

func (cid ContactIdentity) Fingerprint() string {
	return Fingerprint(cid.Key().Public().(ed25519.PublicKey))
}

func (cid ContactIdentity) Sign(data []byte) []byte {
	return Sign(ed25519.PrivateKey(cid), data)
}

func (cid ContactIdentity) Key() ed25519.PrivateKey {
	return ed25519.PrivateKey(cid)
}

func NewContactIdentity() ContactIdentity {
	_, priv, _ := ed25519.GenerateKey(nil)

	return ContactIdentity(priv)
}
