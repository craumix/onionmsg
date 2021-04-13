package types

import (
	"crypto/ed25519"
	"strconv"

	"github.com/wybiral/torgo"
)

type HiddenService struct {
	onion	*torgo.Onion

	Pub 	ed25519.PublicKey
	Priv	ed25519.PrivateKey
}

func NewHiddenService() *HiddenService {
	pub, priv, _ := ed25519.GenerateKey(nil)

	return &HiddenService{
		Pub: pub,
		Priv: priv,
	}
}

func (s *HiddenService) Onion() *torgo.Onion {
	if s.onion == nil {
		s.onion, _ = torgo.OnionFromEd25519(s.Priv)
	}
		
	return s.onion
}

func (s *HiddenService) URL() string {
	return s.Onion().ServiceID + ".onion"
}

func (s *HiddenService) Proxy(torPort int, addr string) {
	s.Onion().Ports[torPort] = addr
}

func (s *HiddenService) LocalProxy(torPort, localPort int) {
	s.Onion().Ports[torPort] = "127.0.0.1:" + strconv.Itoa(localPort)
}