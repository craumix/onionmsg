package daemon

import (
	"log"

	"github.com/craumix/onionmsg/pkg/types"
)

func initExistingContactIDs() (err error) {
	for _, i := range data.ContactIdentities {
		s := i.Service()
		s.LocalProxy(types.PubContPort, loContPort)

		err = torInstance.Controller.AddOnion(s.Onion())
		if err != nil {
			return
		}
	}

	log.Printf("Loaded %d Contact Identities\n", len(data.ContactIdentities))

	return
}

func registerContactIdentity(i *types.Identity) error {
	s := i.Service()
	s.LocalProxy(types.PubContPort, loContPort)

	err := torInstance.Controller.AddOnion(s.Onion())
	if err != nil {
		return err
	}

	data.ContactIdentities = append(data.ContactIdentities, i)

	log.Printf("Registered contact identity %s\n", i.Fingerprint())

	return nil
}

func deregisterContactIdentity(fingerprint string) error {
	i, ok := GetContactID(fingerprint)
	if !ok {
		return nil
	}

	err := torInstance.Controller.DeleteOnion(i.Service().Onion().ServiceID)
	if err != nil {
		return err
	}

	deleteContactIDFromSlice(i)

	log.Printf("Deregistered contact identity %s\n", i.Fingerprint())

	return nil
}
