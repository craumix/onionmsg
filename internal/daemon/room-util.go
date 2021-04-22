package daemon

import (
	"log"

	"github.com/Craumix/onionmsg/internal/types"
	"github.com/google/uuid"
)

func loadRooms() (err error) {
	for _, i := range data.Rooms {
		s := i.Self.Service
		s.LocalProxy(conversationPort, conversationPort)

		err = torInstance.Controller.AddOnion(s.Onion())
		if err != nil {
			return
		}
	}

	log.Printf("Loaded %d Rooms\n", len(data.Rooms))

	for _, room := range data.Rooms {
		room.RunRemoteMessageQueues(torInstance.Proxy, conversationPort)
	}

	return
}

func registerRoom(room *types.Room) error {
	service := room.Self.Service
	service.LocalProxy(conversationPort, conversationPort)

	err := torInstance.Controller.AddOnion(service.Onion())
	if err != nil {
		return err
	}

	data.Rooms[room.ID] = room

	log.Printf("Registered Room %s\n", room.ID)

	room.RunRemoteMessageQueues(torInstance.Proxy, conversationPort)

	return nil
}

func deregisterRoom(id uuid.UUID) error {
	if data.Rooms[id] == nil {
		return nil
	}

	room := data.Rooms[id]
	err := torInstance.Controller.DeleteOnion(room.Self.Service.Onion().ServiceID)
	if err != nil {
		return err
	}

	delete(data.Rooms, id)

	log.Printf("Deregistered Room %s\n", id)

	return nil
}
