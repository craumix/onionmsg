package types

import (
	"crypto/ed25519"
	"encoding/base64"

	"github.com/wybiral/torgo"
)

type RemoteIdentity struct {
	Pub   ed25519.PublicKey `json:"public_key"`
	Nick  string            `json:"nick"`
	Admin bool              `json:"admin"`
}

func NewRemoteIdentity(fingerprint string) (RemoteIdentity, error) {
	raw, err := base64.RawURLEncoding.DecodeString(fingerprint)
	if err != nil {
		return RemoteIdentity{}, err
	}

	return RemoteIdentity{
		Pub: ed25519.PublicKey(raw),
	}, nil
}

func (i *RemoteIdentity) Verify(msg, sig []byte) bool {
	return ed25519.Verify(i.Pub, msg, sig)
}

func (i *RemoteIdentity) URL() string {
	return i.ServiceID() + ".onion"
}

func (i *RemoteIdentity) Fingerprint() string {
	return i.B64PubKey()
}

func (i *RemoteIdentity) B64PubKey() string {
	return base64.RawURLEncoding.EncodeToString(i.Pub)
}

func (i *RemoteIdentity) ServiceID() (id string) {
	id, _ = torgo.ServiceIDFromEd25519(i.Pub)
	return
}
