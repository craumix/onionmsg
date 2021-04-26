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
	Key ed25519.PrivateKey `json:"key"`

	service *HiddenService
}

func NewIdentity() *Identity {
	_, priv, _ := ed25519.GenerateKey(nil)

	return &Identity{
		Key: priv,
	}
}

func (i *Identity) Fingerprint() string {
	return i.B64PubKey()
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
	return base64.RawURLEncoding.EncodeToString(i.Key.Public().(ed25519.PublicKey))
}

func (i *Identity) Sign(data []byte) []byte {
	return ed25519.Sign(i.Key, data)
}

func (i *Identity) Service() *HiddenService {
	if i.service == nil {
		i.service = NewHiddenServiceFromKey(i.Key)
	}
	return i.service
}
