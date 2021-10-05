package daemon

import (
	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
)

func initContIDServices() error {
	for _, i := range data.ContactIdentities {
		err := serveContIDService(i)
		if err != nil {
			return err
		}
	}

	return nil
}

func registerContID(id types.Identity) error {
	err := serveContIDService(id)
	if err != nil {
		return err
	}

	data.ContactIdentities = append(data.ContactIdentities, id)
	log.Infof("Registered contact identity %s\n", id.Fingerprint())

	return nil
}

func serveContIDService(id types.Identity) error {
	return torInstance.RegisterService(*id.Priv, types.PubContPort, loContPort)
}

func deregisterContID(fingerprint string) error {
	i, ok := GetContactID(fingerprint)
	if !ok {
		return nil
	}

	err := torInstance.DeregisterService(*i.Pub)
	if err != nil {
		return err
	}

	deleteContactIDFromSlice(i)

	log.Debugf("Deregistered contact identity %s\n", i.Fingerprint())

	return nil
}
