package daemon

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
	uid "github.com/google/uuid"
)

//GetTorlog returns the log of the used to instance.
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

//ListContactIDs returns a list of all the contactid's fingerprints.
func ListContactIDs() []string {
	contIDs := make([]string, 0)
	for _, id := range data.ContactIdentities {
		contIDs = append(contIDs, id.Fingerprint())
	}
	return contIDs
}

//ListRooms returns a list of all the rooms uuid's formated as strings.
func ListRooms() []string {
	rooms := make([]string, 0)
	for _, r := range data.Rooms {
		rooms = append(rooms, r.ID.String())
	}
	return rooms
}

//CreateContactID generates and registers a new contact id and returns its fingerprint.
func CreateContactID() (string, error) {
	id := types.NewIdentity()
	err := registerContactIdentity(id)
	if err != nil {
		return "", err
	}
	return id.Fingerprint(), nil
}

//DeleteContactID deletes and deregisters a contact id.
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
	room, ok := GetRoom(roomID)
	if !ok {
		return fmt.Errorf("no such room %s", roomID)
	}

	id, err := types.NewRemoteIdentity(fingerprint)
	if err != nil {
		return err
	}

	return room.AddUser(id, torInstance.Proxy, contactPort)
}

//DeleteRoom deletes the room with the specified uuid
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

	room, ok := GetRoom(id)
	if !ok {
		return fmt.Errorf("no such room: %s", uuid)
	}

	return room.SendMessage(msgType, content)
}

func ListMessages(uuid string) ([]*types.Message, error) {
	id, err := uid.Parse(uuid)
	if err != nil {
		return nil, err
	}

	room, ok := GetRoom(id)
	if !ok {
		return nil, fmt.Errorf("no such room: %s", uuid)
	}
	return room.Messages, nil
}


func GetRoom(id uuid.UUID) (*types.Room, bool) {
	for _, r := range data.Rooms {
		if r.ID == id {
			return r, true
		}
	}
	return nil, false
}

func GetContactID(fingerprint string) (*types.Identity, bool) {
	for _, i := range data.ContactIdentities {
		if i.Fingerprint() == fingerprint {
			return i, true
		}
	}
	return nil, false
}

func deleteRoomFromSlice(item *types.Room) {
	var i int
	for j, e := range data.Rooms {
		if e == item {
			i = j
		}
	}

	data.Rooms[len(data.Rooms)-1], data.Rooms[i] = data.Rooms[i], data.Rooms[len(data.Rooms)-1]
	data.Rooms = data.Rooms[:len(data.Rooms) - 1]
}

func deleteContactIDFromSlice(item *types.Identity) {
	var i int
	for j, e := range data.ContactIdentities {
		if e == item {
			i = j
		}
	}

	data.ContactIdentities[len(data.ContactIdentities)-1], data.ContactIdentities[i] = data.ContactIdentities[i], data.ContactIdentities[len(data.ContactIdentities)-1]
	data.ContactIdentities = data.ContactIdentities[:len(data.ContactIdentities) - 1]
}