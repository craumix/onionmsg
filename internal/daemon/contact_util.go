package daemon

import (
	"log"

	"github.com/craumix/onionmsg/pkg/types"
)

func initContIDServices() error {
	for _, i := range data.ContactIdentities {
		err := serveContIDService(i)
		if err != nil {
			return err
		}
	}

	log.Printf("Loaded %d Contact Identities\n", len(data.ContactIdentities))

	return nil
}

func registerContID(id types.Identity) error {
	err := serveContIDService(id)
	if err != nil {
		return err
	}

	data.ContactIdentities = append(data.ContactIdentities, id)
	log.Printf("Registered contact identity %s\n", id.Fingerprint())

	return nil
}

func serveContIDService(id types.Identity) error {
	return torInstance.RegisterService(id, types.PubContPort, loContPort)
}

func deregisterContID(fingerprint string) error {
	i, ok := GetContactID(fingerprint)
	if !ok {
		return nil
	}

	err := torInstance.DeregisterService(i)
	if err != nil {
		return err
	}

	deleteContactIDFromSlice(i)

	log.Printf("Deregistered contact identity %s\n", i)

	return nil
}
