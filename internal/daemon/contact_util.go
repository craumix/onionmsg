package daemon

import (
	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
)

func (d *Daemon) CreateContactID() (types.ContactIdentity, error) {
	cID := types.NewContactIdentity()

	err := d.registerContactID(cID)
	if err != nil {
		return types.ContactIdentity{}, err
	}

	return cID, nil
}

func (d *Daemon) DeleteContactID(fingerprint types.Fingerprint) error {
	err := d.deregisterContactID(fingerprint)
	if err != nil {
		return err
	}

	d.removeContactID(fingerprint)

	return nil
}

func (d *Daemon) GetContactIDs() []types.ContactIdentity {
	return d.data.ContactIdentities
}

func (d *Daemon) initContactIDServices() error {
	for _, i := range d.GetContactIDs() {
		err := d.serveContactIDService(i)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Daemon) registerContactID(cID types.ContactIdentity) error {
	err := d.serveContactIDService(cID)
	if err != nil {
		return err
	}

	d.addContactID(cID)

	log.WithField("fingerprint", cID.Fingerprint()).Info("registered contact identity")

	return nil
}

func (d *Daemon) serveContactIDService(id types.ContactIdentity) error {
	return d.Tor.RegisterService(id.Priv, d.loContPort, d.loContPort)
}

func (d *Daemon) deregisterContactID(fingerprint types.Fingerprint) error {
	i, ok := d.getContactID(fingerprint)
	if !ok {
		return nil
	}

	err := d.Tor.DeregisterService(i.Pub)
	if err != nil {
		return err
	}

	log.WithField("fingerprint", i.Fingerprint()).Debugf("deregistered contact identity")

	return nil
}

func (d *Daemon) addContactID(cID types.ContactIdentity) {
	d.data.ContactIdentities = append(d.data.ContactIdentities, cID)
}

func (d *Daemon) removeContactID(toRemove types.Fingerprint) {
	for i, cID := range d.GetContactIDs() {
		if cID.Fingerprint() == toRemove {
			d.data.ContactIdentities = append(d.data.ContactIdentities[:i], d.data.ContactIdentities[i+1:]...)
			return
		}
	}
}

func (d *Daemon) getContactID(toFind types.Fingerprint) (types.ContactIdentity, bool) {
	for _, cID := range d.GetContactIDs() {
		if cID.Fingerprint() == toFind {
			return cID, true
		}
	}

	return types.ContactIdentity{}, false
}
