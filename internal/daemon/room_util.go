package daemon

import (
	"log"

	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
)

func initRooms() (err error) {
	for _, r := range data.Rooms {
		err = serveConvIDService(r.Self)
		if err != nil {
			return
		}
	}

	log.Printf("Loaded %d Rooms\n", len(data.Rooms))

	for _, room := range data.Rooms {
		room.RunRemoteMessageQueues(torInstance.Proxy)
	}

	return
}

func registerRoom(room *types.Room) error {
	err := serveConvIDService(room.Self)
	if err != nil {
		return err
	}

	data.Rooms = append(data.Rooms, room)
	log.Printf("Registered Room %s\n", room.ID)

	return nil
}

func serveConvIDService(i types.Identity) error {
	return torInstance.RegisterService(i.Key, types.PubConvPort, loConvPort)
}

func deregisterRoom(id uuid.UUID) error {
	r, ok := GetRoom(id)
	if !ok {
		return nil
	}

	err := torInstance.DeregisterService(r.Self.Key)
	if err != nil {
		return err
	}

	r.StopQueues()

	deleteRoomFromSlice(r)

	log.Printf("Deregistered Room %s\n", id)

	return nil
}
