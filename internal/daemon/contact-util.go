package daemon

import (
	"log"

	"github.com/craumix/onionmsg/internal/types"
)

func loadContactIdentites() (err error) {
	for _, i := range data.ContactIdentities {
		s := i.Service()
		s.LocalProxy(contactPort, contactPort)

		err = torInstance.Controller.AddOnion(s.Onion())
		if err != nil {
			return
		}
	}

	log.Printf("Loaded %d Contact Identities\n", len(data.ContactIdentities))

	return
}

func registerContactIdentity(i *types.Identity) error {
	service := i.Service()
	service.LocalProxy(contactPort, contactPort)

	err := torInstance.Controller.AddOnion(service.Onion())
	if err != nil {
		return err
	}

	data.ContactIdentities[i.Fingerprint()] = i

	log.Printf("Registered contact identity %s\n", i.Fingerprint())

	return nil
}

func deregisterContactIdentity(fingerprint string) error {
	if data.ContactIdentities[fingerprint] == nil {
		return nil
	}

	i := data.ContactIdentities[fingerprint]
	err := torInstance.Controller.DeleteOnion(i.Service().Onion().ServiceID)
	if err != nil {
		return err
	}

	delete(data.ContactIdentities, fingerprint)

	log.Printf("Deregistered contact identity %s\n", i.Fingerprint())

	return nil
}
