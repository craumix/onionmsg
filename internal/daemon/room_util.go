package daemon

import (
	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
	"github.com/google/uuid"
)

func (d *Daemon) initRooms() error {
	for _, r := range d.data.Rooms {
		err := d.serveConvIDService(r.Self)
		if err != nil {
			return err
		}
	}

	for _, room := range d.data.Rooms {
		room.RunMessageQueueForAllPeers()
	}

	return nil
}

func (d *Daemon) registerRoom(room *types.Room) error {
	err := d.serveConvIDService(room.Self)
	if err != nil {
		return err
	}

	d.data.Rooms = append(d.data.Rooms, room)
	log.WithField("room", room.ID.String()).Info("registered room")

	d.Notifier.NotifyNewRoom(room.Info())

	return nil
}

func (d *Daemon) serveConvIDService(i types.Identity) error {
	return d.Tor.RegisterService(*i.Priv, types.PubConvPort, d.loConvPort)
}

func (d *Daemon) deregisterRoom(id uuid.UUID) error {
	r, ok := d.GetRoom(id)
	if !ok {
		return nil
	}

	err := d.Tor.DeregisterService(*r.Self.Pub)
	if err != nil {
		return err
	}

	r.StopQueues()

	d.DeleteRoomFromSlice(r)

	log.WithField("room", id.String()).Info("degistered room")

	return nil
}
