package daemon

import (
	"context"
	"fmt"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
	uid "github.com/google/uuid"
)

var (
	TorInfo = getTorInfo

	ListContactIDs  = listContactIDs
	CreateContactID = createContactID
	DeleteContact   = DeleteContactID

	Rooms         = listRooms
	CreateRoom    = createRoom
	DeleteRoom    = deleteRoom
	AddPeerToRoom = addPeerToRoom
	ListMessages  = listMessages

	SendMessage = sendMessage
)

// GetTorlog returns the log of the used to instance.
func getTorInfo() interface{} {
	return struct {
		Log        string `json:"log"`
		Version    string `json:"version"`
		PID        int    `json:"pid"`
		BinaryPath string `json:"path"`
	}{
		torInstance.Log(),
		torInstance.Version(),
		torInstance.Pid(),
		torInstance.BinaryPath(),
	}
}

// listContactIDs returns a list of all the contactId's fingerprints.
func listContactIDs() []string {
	var contIDs []string
	for _, id := range data.ContactIdentities {
		contIDs = append(contIDs, id.Fingerprint())
	}
	return contIDs
}

// listRooms returns a marshaled list of all the rooms with most information
func listRooms() []*types.RoomInfo {
	var rooms []*types.RoomInfo
	for _, r := range data.Rooms {
		rooms = append(rooms, r.Info())
	}

	return rooms
}

// createContactID generates and registers a new contact id and returns its fingerprint.
func createContactID() (string, error) {
	id := types.NewIdentity()
	err := registerContID(id)
	if err != nil {
		return "", err
	}
	return id.Fingerprint(), nil
}

// DeleteContactID deletes and deregisters a contact id.
func DeleteContactID(fingerprint string) error {
	return deregisterContID(fingerprint)
}

// Maybe this should be run in a goroutine
func createRoom(fingerprints []string) error {
	var ids []types.RemoteIdentity
	for _, fingerprint := range fingerprints {
		id, err := types.NewRemoteIdentity(fingerprint)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	// TODO derive this from an actual context
	room, err := types.NewRoom(context.Background(), ids...)
	if err != nil {
		return err
	}

	return registerRoom(room)
}

// Maybe this should be run in a goroutine
func addPeerToRoom(roomID uuid.UUID, fingerprint string) error {
	room, ok := GetRoom(roomID)
	if !ok {
		return fmt.Errorf("no such room %s", roomID)
	}

	id, err := types.NewRemoteIdentity(fingerprint)
	if err != nil {
		return err
	}

	return room.AddPeers(id)
}

// deleteRoom deletes the room with the specified uuid.
func deleteRoom(uuid string) error {
	id, err := uid.Parse(uuid)
	if err != nil {
		return err
	}
	return deregisterRoom(id)
}

func sendMessage(uuid string, msgType types.MessageType, content []byte) error {
	id, err := uid.Parse(uuid)
	if err != nil {
		return err
	}

	room, ok := GetRoom(id)
	if !ok {
		return fmt.Errorf("no such room: %s", uuid)
	}

	return room.SendMessageToAllPeers(msgType, content)
}

func listMessages(uuid string, count int) ([]types.Message, error) {
	id, err := uid.Parse(uuid)
	if err != nil {
		return nil, err
	}

	room, ok := GetRoom(id)
	if !ok {
		return nil, fmt.Errorf("no such room: %s", uuid)
	}

	if count > 0 {
		return room.Messages[len(room.Messages)-count:], nil
	} else {
		return room.Messages, nil
	}
}

func GetRoom(id uuid.UUID) (*types.Room, bool) {
	for _, r := range data.Rooms {
		if r.ID == id {
			return r, true
		}
	}
	return nil, false
}

func GetContactID(fingerprint string) (types.Identity, bool) {
	for _, i := range data.ContactIdentities {
		if i.Fingerprint() == fingerprint {
			return i, true
		}
	}
	return types.Identity{}, false
}

func deleteRoomFromSlice(item *types.Room) {
	var i int
	for j, e := range data.Rooms {
		if e == item {
			i = j
		}
	}

	data.Rooms[len(data.Rooms)-1], data.Rooms[i] = data.Rooms[i], data.Rooms[len(data.Rooms)-1]
	data.Rooms = data.Rooms[:len(data.Rooms)-1]
}

func deleteContactIDFromSlice(item types.Identity) {
	var i int
	for j, e := range data.ContactIdentities {
		if e.Fingerprint() == item.Fingerprint() {
			i = j
			break
		}
	}

	data.ContactIdentities[len(data.ContactIdentities)-1], data.ContactIdentities[i] = data.ContactIdentities[i], data.ContactIdentities[len(data.ContactIdentities)-1]
	data.ContactIdentities = data.ContactIdentities[:len(data.ContactIdentities)-1]
}
