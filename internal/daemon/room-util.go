package daemon

import (
	"log"

	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
)

func initExistingRooms() (err error) {
	for _, i := range data.Rooms {
		s := i.Self.Service()
		s.LocalProxy(types.PubConvPort, loConvPort)

		err = torInstance.Controller.AddOnion(s.Onion())
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
	s := room.Self.Service()
	s.LocalProxy(types.PubConvPort, loConvPort)

	err := torInstance.Controller.AddOnion(s.Onion())
	if err != nil {
		return err
	}

	data.Rooms = append(data.Rooms, room)

	log.Printf("Registered Room %s\n", room.ID)

	return nil
}

func deregisterRoom(id uuid.UUID) error {
	room, ok := GetRoom(id)
	if !ok {
		return nil
	}
	err := torInstance.Controller.DeleteOnion(room.Self.Service().Onion().ServiceID)
	if err != nil {
		return err
	}

	room.StopQueues()

	deleteRoomFromSlice(room)

	log.Printf("Deregistered Room %s\n", id)

	return nil
}
