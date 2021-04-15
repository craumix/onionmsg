package types

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"image"
	"image/png"

	qrcode "github.com/skip2/go-qrcode"
)

type Identity struct {
	Service *HiddenService		`json:"hidden_service"`
	Pub 	ed25519.PublicKey	`json:"public_key"`
	Priv	ed25519.PrivateKey	`json:"private_key"`
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

func (i *Identity) QR(res int) (image.Image, error) {
	b, err := qrcode.Encode(i.Fingerprint(), qrcode.Medium, res)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (i *Identity) B64PubKey() string {
	return base64.RawURLEncoding.EncodeToString(i.Pub)
}

func (i *Identity) Sign(data []byte) []byte {
	return ed25519.Sign(i.Priv, data)
}