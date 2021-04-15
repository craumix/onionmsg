package types

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

type RemoteIdentity struct {
	Pub		ed25519.PublicKey	`json:"public_key"`
	Service	string				`json:"service"`
}

func NewRemoteIdentity(fingerprint string) (*RemoteIdentity, error) {
	if !strings.Contains(fingerprint, "@") {
		return nil, fmt.Errorf("%s is not a valid id", fingerprint)
	}

	tmp := strings.Split(fingerprint, "@")
	k, err := base64.RawURLEncoding.DecodeString(tmp[0])
	if err != nil {
		return nil, err
	}

	return &RemoteIdentity{
		Pub: ed25519.PublicKey(k),
		Service: tmp[1],
	}, nil
}

func (i *RemoteIdentity) Verify(msg, sig []byte) bool {
	return ed25519.Verify(i.Pub, msg, sig)
}

func (i *RemoteIdentity) URL() string {
	return i.Service + ".onion"
}

func (i *RemoteIdentity) Fingerprint() string {
	return i.B64PubKey() + "@" + i.Service
}

func (i *RemoteIdentity) B64PubKey() string {
	return base64.RawURLEncoding.EncodeToString(i.Pub)
}