package daemon

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
	uid "github.com/google/uuid"
)

func GetTorlog() (string, error) {
	logfile, err := os.OpenFile(tordir+"/tor.log", os.O_RDONLY, 0600)
	if err != nil {
		return "", err
	}

	logs, err := ioutil.ReadAll(logfile)
	if err != nil {
		return "", err
	}
	return string(logs), nil
}

func ListContactIDs() []string {
	contIDs := make([]string, 0)
	for _, id := range data.ContactIdentities {
		contIDs = append(contIDs, id.Fingerprint())
	}
	return contIDs
}

func ListRooms() []string {
	rooms := make([]string, 0)
	for key := range data.Rooms {
		rooms = append(rooms, key.String())
	}
	return rooms
}

func CreateContactID() (string, error) {
	id := types.NewIdentity()
	err := registerContactIdentity(id)
	if err != nil {
		return "", err
	}
	return id.Fingerprint(), nil
}

func DeleteContactID(fingerprint string) error {
	return deregisterContactIdentity(fingerprint)
}

// Maybe this should be run in a goroutine
func CreateRoom(fingerprints []string) error {
	var ids []*types.RemoteIdentity
	for _, fingerprint := range fingerprints {
		id, err := types.NewRemoteIdentity(fingerprint)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	room, err := types.NewRoom(ids, torInstance.Proxy, contactPort, conversationPort)
	if err != nil {
		return err
	}

	return registerRoom(room)
}

// Maybe this should be run in a goroutine
func AddUserToRoom(roomID uuid.UUID, fingerprint string) error {
	if data.Rooms[roomID] == nil {
		return fmt.Errorf("no such room %s", roomID)
	}

	id, err := types.NewRemoteIdentity(fingerprint)
	if err != nil {
		return err
	}

	return data.Rooms[roomID].AddUser(id, torInstance.Proxy, contactPort)
}

func DeleteRoom(uuid string) error {
	id, err := uid.Parse(uuid)
	if err != nil {
		return err
	}
	return deregisterRoom(id)
}

func SendMessage(uuid string, msgType byte, content []byte) error {
	id, err := uid.Parse(uuid)
	if err != nil {
		return err
	}

	room, ok := data.Rooms[id]
	if !ok {
		return fmt.Errorf("No such room: %s", uuid)
	}

	return room.SendMessage(msgType, content)
}

func ListMessages(uuid string) ([]*types.Message, error) {
	id, err := uid.Parse(uuid)
	if err != nil {
		return nil, err
	}

	room, ok := data.Rooms[id]
	if !ok {
		return nil, fmt.Errorf("No such room: %s", uuid)
	}
	return room.Messages, nil
}
