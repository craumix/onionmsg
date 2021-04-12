package types

import (
	"crypto/ed25519"

	"github.com/wybiral/torgo"
)

func NewHiddenService() *HiddenService {
	_, priv, _ := ed25519.GenerateKey(nil)

	return &HiddenService{
		Key: priv,
	}
}

type HiddenService struct {
	Key		ed25519.PrivateKey

	onion	*torgo.Onion
}

func (s *HiddenService) Onion() *torgo.Onion {
	if s.onion == nil {
		s.onion, _ = torgo.OnionFromEd25519(s.Key)
	}
		
	return s.onion
}

func (s *HiddenService) URL() string {
	return s.Onion().ServiceID + ".onion"
}