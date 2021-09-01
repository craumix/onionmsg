package daemon

import (
	"crypto/ed25519"
	"log"

	"github.com/craumix/onionmsg/pkg/types"
)

func initContIDServices() error {
	for _, key := range data.Keys {
		err := serveContIDService(key)
		if err != nil {
			return err
		}
	}

	log.Printf("Loaded %d Contact Identities\n", len(data.Keys))

	return nil
}

func registerContID(key ed25519.PrivateKey) error {
	err := serveContIDService(key)
	if err != nil {
		return err
	}

	data.Keys = append(data.Keys, key)
	log.Printf("Registered contact identity %s\n", types.Fingerprint(key))

	return nil
}

func serveContIDService(key ed25519.PrivateKey) error {
	return torInstance.RegisterService(key, types.PubContPort, loContPort)
}

func deregisterContID(fingerprint string) error {
	key, ok := GetKey(fingerprint)
	if !ok {
		return nil
	}

	err := torInstance.DeregisterService(key)
	if err != nil {
		return err
	}

	deleteKeyFromSlice(types.Fingerprint(key))

	log.Printf("Deregistered contact identity %s\n", types.Fingerprint(key))

	return nil
}
