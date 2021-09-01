package daemon

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
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
	for _, key := range data.Keys {
		contIDs = append(contIDs, types.Fingerprint(key))
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
	key := types.GenerateKey()
	err := registerContID(key)
	if err != nil {
		return "", err
	}
	return types.Fingerprint(key), nil
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
func deleteRoom(uid string) error {
	id, err := uuid.Parse(uid)
	if err != nil {
		return err
	}
	return deregisterRoom(id)
}

func sendMessage(uid string, content types.MessageContent) error {
	id, err := uuid.Parse(uid)
	if err != nil {
		return err
	}

	room, ok := GetRoom(id)
	if !ok {
		return fmt.Errorf("no such room: %s", uid)
	}

	room.SendMessageToAllPeers(content)
	return nil
}

func listMessages(uid string, count int) ([]types.Message, error) {
	id, err := uuid.Parse(uid)
	if err != nil {
		return nil, err
	}

	room, ok := GetRoom(id)
	if !ok {
		return nil, fmt.Errorf("no such room: %s", uid)
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

func GetKey(fingerprint string) (ed25519.PrivateKey, bool) {
	for _, key := range data.Keys {
		if types.Fingerprint(key) == fingerprint {
			return key, true
		}
	}
	return ed25519.PrivateKey{}, false
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

func deleteKeyFromSlice(fingerprint string) {
	i := 0
	for ; i < len(data.Keys); i++ {
		if types.Fingerprint(data.Keys[i]) == fingerprint {
			break
		}
	}

	data.Keys[len(data.Keys)-1], data.Keys[i] = data.Keys[i], data.Keys[len(data.Keys)-1]
	data.Keys = data.Keys[:len(data.Keys)-1]
}
