package daemon

import (
	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
	"github.com/google/uuid"
)

func initRooms() (err error) {
	for _, r := range data.Rooms {
		err = serveConvIDService(r.Self)
		if err != nil {
			return
		}
	}

	for _, room := range data.Rooms {
		room.RunMessageQueueForAllPeers()
	}

	return
}

func registerRoom(room *types.Room) error {
	err := serveConvIDService(room.Self)
	if err != nil {
		return err
	}

	data.Rooms = append(data.Rooms, room)
	log.WithField("room", room.ID.String()).Info("registered room")

	notifyNewRoom(room.Info())

	return nil
}

func serveConvIDService(i types.Identity) error {
	return torInstance.RegisterService(*i.Priv, types.PubConvPort, loConvPort)
}

func deregisterRoom(id uuid.UUID) error {
	r, ok := GetRoom(id)
	if !ok {
		return nil
	}

	err := torInstance.DeregisterService(*r.Self.Pub)
	if err != nil {
		return err
	}

	r.StopQueues()

	deleteRoomFromSlice(r)

	log.WithField("room", id.String()).Info("degistered room")

	return nil
}
