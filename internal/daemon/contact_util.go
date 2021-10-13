package daemon

import (
	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
)

func (d *Daemon) initContactIDServices() error {
	for _, i := range d.GetContactIDs() {
		err := d.serveContactIDService(i)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Daemon) registerContactID(cID types.Identity) error {
	err := d.serveContactIDService(cID)
	if err != nil {
		return err
	}

	d.AddContactID(cID)

	log.WithField("fingerprint", cID.Fingerprint()).Info("registered contact identity")

	return nil
}

func (d *Daemon) serveContactIDService(id types.Identity) error {
	return d.Tor.RegisterService(*id.Priv, types.PubContPort, d.loContPort)
}

func (d *Daemon) deregisterContactID(fingerprint string) error {
	i, ok := d.GetContactIDByFingerprint(fingerprint)
	if !ok {
		return nil
	}

	err := d.Tor.DeregisterService(*i.Pub)
	if err != nil {
		return err
	}

	log.WithField("fingerprint", i.Fingerprint()).Debugf("deregistered contact identity")

	return nil
}

func (d *Daemon) GetContactIDsAsStrings() []string {
	var cIDs []string
	for _, cID := range d.GetContactIDs() {
		cIDs = append(cIDs, cID.Fingerprint())
	}
	return cIDs
}

func (d *Daemon) CreateAndRegisterNewContactID() (types.Identity, error) {
	cID, _ := types.NewIdentity(types.Contact, "")

	err := d.registerContactID(cID)
	if err != nil {
		return types.Identity{}, err
	}

	return cID, nil
}

func (d *Daemon) DeregisterAndRemoveContactIDByFingerprint(fingerprint string) error {
	err := d.deregisterContactID(fingerprint)
	if err != nil {
		return err
	}

	d.RemoveContactIDByFingerprint(fingerprint)

	return nil
}

func (d *Daemon) GetContactIDs() []types.Identity {
	return d.data.ContactIdentities
}

func (d *Daemon) AddContactID(cID types.Identity) {
	if cID.Type == types.Contact {
		d.data.ContactIdentities = append(d.data.ContactIdentities, cID)
	}
}

func (d *Daemon) RemoveContactID(toRemove types.Identity) {
	d.RemoveContactIDByFingerprint(toRemove.Fingerprint())
}

func (d *Daemon) RemoveContactIDByFingerprint(toRemove string) {
	for i, cID := range d.GetContactIDs() {
		if cID.Fingerprint() == toRemove {
			d.data.ContactIdentities = append(d.data.ContactIdentities[:i], d.data.ContactIdentities[i+1:]...)
			return
		}
	}
}

func (d *Daemon) GetContactIDByFingerprint(toFind string) (types.Identity, bool) {
	for _, cID := range d.GetContactIDs() {
		if cID.Fingerprint() == toFind {
			return cID, true
		}
	}

	return types.Identity{}, false
}
