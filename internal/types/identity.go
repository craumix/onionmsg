package types

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"github.com/wybiral/torgo"
)

type identityPub struct {
	Pub ed25519.PublicKey `json:"pub"`
}

func (i identityPub) Verify(msg, sig []byte) (bool, error) {
	return ed25519.Verify(i.Pub, msg, sig), nil
}

func (i identityPub) Fingerprint() string {
	return base64.RawURLEncoding.EncodeToString(i.Pub)
}

func (i identityPub) ServiceID() string {
	id, _ := torgo.ServiceIDFromEd25519(i.Pub)
	return id
}

func (i identityPub) URL() string {
	return i.ServiceID() + ".onion"
}

type identityPriv struct {
	Priv ed25519.PrivateKey `json:"priv"`
}

func (i identityPriv) Sign(data []byte) ([]byte, error) {
	return ed25519.Sign(i.Priv, data), nil
}

type identityMeta struct {
	Nickname string `json:"nick"`
	Admin    bool   `json:"admin"`
}

func (i identityMeta) Nick() string {
	return i.Nickname
}

func (i *identityMeta) SetNick(nick string) {
	i.Nickname = nick
}

func (i identityMeta) isAdmin() bool {
	return i.Admin
}

func (i *identityMeta) SetAdmin(isAdmin bool) {
	i.Admin = isAdmin
}

type RemoteIdentity struct {
	identityMeta
	identityPub
}

func (i RemoteIdentity) String() string {
	return fmt.Sprintf("Remote: %s", i.Fingerprint())
}

func NewRemoteIdentity(fingerprint string) (RemoteIdentity, error) {
	rid := RemoteIdentity{}

	var err error
	rid.Pub, err = getPubKeyFromFingerprint(fingerprint)
	if err != nil {
		return RemoteIdentity{}, err
	}

	return rid, nil
}

type SelfIdentity struct {
	identityMeta
	identityPub
	identityPriv
}

func (i SelfIdentity) String() string {
	return fmt.Sprintf("Self: %s", i.Fingerprint())
}

func NewSelfIdentity() SelfIdentity {
	sid := SelfIdentity{}

	sid.Pub, sid.Priv = generateKeyPair()

	return sid
}

type ContactIdentity struct {
	identityPub
	identityPriv
}

func (i ContactIdentity) String() string {
	return fmt.Sprintf("Contact: %s", i.Fingerprint())
}
func NewContactIdentity() ContactIdentity {
	cid := ContactIdentity{}

	cid.Pub, cid.Priv = generateKeyPair()

	return cid
}

func getPubKeyFromFingerprint(fingerprint string) (ed25519.PublicKey, error) {
	raw, err := base64.RawURLEncoding.DecodeString(fingerprint)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func generateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey) {
	_, privKey, _ := ed25519.GenerateKey(nil)

	pubKey := privKey.Public().(ed25519.PublicKey)

	return pubKey, privKey
}
