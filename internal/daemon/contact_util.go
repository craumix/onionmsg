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

func registerContID(i types.Identity) error {
	err := serveContIDService(i)
	if err != nil {
		return err
	}

	data.ContactIdentities = append(data.ContactIdentities, i)
	log.Printf("Registered contact identity %s\n", i.Fingerprint())

	return nil
}

func serveContIDService(i types.Identity) error {
	return torInstance.RegisterService(i.Key, types.PubContPort, loContPort)
}

func deregisterContID(fingerprint string) error {
	i, ok := GetContactID(fingerprint)
	if !ok {
		return nil
	}

	err := torInstance.DeregisterService(i.Key)
	if err != nil {
		return err
	}

	deleteContactIDFromSlice(i)

	log.Printf("Deregistered contact identity %s\n", i.Fingerprint())

	return nil
}
