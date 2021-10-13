package daemon

import (
	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
)

func (d *Daemon) initContIDServices() error {
	for _, i := range d.data.ContactIdentities {
		err := d.serveContIDService(i)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Daemon) registerContID(id types.Identity) error {
	err := d.serveContIDService(id)
	if err != nil {
		return err
	}

	d.data.ContactIdentities = append(d.data.ContactIdentities, id)
	log.WithField("fingerprint", id.Fingerprint()).Info("registered contact identity")

	return nil
}

func (d *Daemon) serveContIDService(id types.Identity) error {
	return d.Tor.RegisterService(*id.Priv, types.PubContPort, d.loContPort)
}

func (d *Daemon) deregisterContID(fingerprint string) error {
	i, ok := d.GetContactID(fingerprint)
	if !ok {
		return nil
	}

	err := d.Tor.DeregisterService(*i.Pub)
	if err != nil {
		return err
	}

	d.DeleteContactIDFromSlice(i)

	log.WithField("fingerprint", i.Fingerprint()).Debugf("deregistered contact identity")

	return nil
}
