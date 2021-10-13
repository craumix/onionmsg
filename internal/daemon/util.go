package daemon

import (
	"context"
	"fmt"
	"strings"

	"github.com/craumix/onionmsg/internal/types"
	"github.com/google/uuid"
)

type StringWriter struct {
	OnWrite func(string)
}

func (w StringWriter) Write(p []byte) (int, error) {
	if w.OnWrite != nil {
		lines := strings.Split(string(p), "\n")
		lines = lines[:len(lines)-1]
		for _, v := range lines {
			w.OnWrite(v)
		}
	}
	return len(p), nil
}

// listContactIDs returns a list of all the contactId's fingerprints.
func (d *Daemon) ListContactIDs() []string {
	var contIDs []string
	for _, id := range d.data.ContactIdentities {
		contIDs = append(contIDs, id.Fingerprint())
	}
	return contIDs
}

// listRooms returns a marshaled list of all the rooms with most information
func (d *Daemon) ListRooms() []*types.RoomInfo {
	var rooms []*types.RoomInfo
	for _, r := range d.data.Rooms {
		rooms = append(rooms, r.Info())
	}

	return rooms
}

func (d *Daemon) RoomInfo(id uuid.UUID) (*types.RoomInfo, error) {
	for _, r := range d.data.Rooms {
		if r.ID == id {
			return r.Info(), nil
		}
	}

	return nil, fmt.Errorf("room with id %s doesn't exist", id)
}

// createContactID generates and registers a new contact id and returns its fingerprint.
func (d *Daemon) CreateContactID() (string, error) {
	id, _ := types.NewIdentity(types.Contact, "")
	err := d.registerContID(id)
	if err != nil {
		return "", err
	}
	return id.Fingerprint(), nil
}

// DeleteContactID deletes and deregisters a contact id.
func (d *Daemon) DeleteContactID(fingerprint string) error {
	return d.deregisterContID(fingerprint)
}

// Maybe this should be run in a goroutine
func (d *Daemon) CreateRoom(fingerprints []string) error {
	var ids []types.Identity
	for _, fingerprint := range fingerprints {
		id, err := types.NewIdentity(types.Remote, fingerprint)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	room, err := types.NewRoom(d.ctx, ids...)
	if err != nil {
		return err
	}

	return d.registerRoom(room)
}

// Maybe this should be run in a goroutine
func (d *Daemon) AddPeerToRoom(roomID uuid.UUID, fingerprint string) error {
	room, found := d.GetRoom(roomID)
	if !found {
		return fmt.Errorf("no such room %s", roomID)
	}

	id, err := types.NewIdentity(types.Remote, fingerprint)
	if err != nil {
		return err
	}

	return room.AddPeers(id)
}

// deleteRoom deletes the room with the specified uuid.
func (d *Daemon) DeleteRoom(uid string) error {
	id, err := uuid.Parse(uid)
	if err != nil {
		return err
	}
	return d.deregisterRoom(id)
}

func (d *Daemon) SendMessage(uid string, content types.MessageContent) error {
	id, err := uuid.Parse(uid)
	if err != nil {
		return err
	}

	room, found := d.GetRoom(id)
	if !found {
		return fmt.Errorf("no such room: %s", uid)
	}

	room.SendMessageToAllPeers(content)
	return nil
}

func (d *Daemon) ListMessages(uid string, count int) ([]types.Message, error) {
	id, err := uuid.Parse(uid)
	if err != nil {
		return nil, err
	}

	room, found := d.GetRoom(id)
	if !found {
		return nil, fmt.Errorf("no such room: %s", uid)
	}

	if count > 0 && count < len(room.Messages) {
		return room.Messages[len(room.Messages)-count:], nil
	} else {
		return room.Messages, nil
	}
}

func (d *Daemon) GetRoom(id uuid.UUID) (*types.Room, bool) {
	for _, r := range d.data.Rooms {
		if r.ID == id {
			return r, true
		}
	}
	return nil, false
}

func (d *Daemon) GetContactID(fingerprint string) (types.Identity, bool) {
	for _, i := range d.data.ContactIdentities {
		if i.Fingerprint() == fingerprint {
			return i, true
		}
	}
	return types.Identity{}, false
}

func (d *Daemon) DeleteRoomFromSlice(item *types.Room) {
	for j, e := range d.data.Rooms {
		if e == item {
			d.data.Rooms[len(d.data.Rooms)-1], d.data.Rooms[j] = d.data.Rooms[j], d.data.Rooms[len(d.data.Rooms)-1]
			d.data.Rooms = d.data.Rooms[:len(d.data.Rooms)-1]
			break
		}
	}

}

func (d *Daemon) DeleteContactIDFromSlice(cid types.Identity) {
	for i := 0; i < len(d.data.ContactIdentities); i++ {
		if d.data.ContactIdentities[i].Fingerprint() == cid.Fingerprint() {
			d.data.ContactIdentities[len(d.data.ContactIdentities)-1], d.data.ContactIdentities[i] = d.data.ContactIdentities[i], d.data.ContactIdentities[len(d.data.ContactIdentities)-1]
			d.data.ContactIdentities = d.data.ContactIdentities[:len(d.data.ContactIdentities)-1]

			break
		}
	}

}

func (d *Daemon) RequestList() []*types.RoomRequest {
	return d.data.Requests
}

func (d *Daemon) AcceptRoomRequest(id uuid.UUID) error {
	for _, v := range d.data.Requests {
		if v.ID == id {
			v.Room.SetContext(context.Background())

			err := d.registerRoom(&v.Room)
			if err != nil {
				return err
			}

			v.Room.RunMessageQueueForAllPeers()

			v.Room.SendMessageToAllPeers(types.MessageContent{
				Type: types.ContentTypeCmd,
				Data: types.ConstructCommand(nil, types.RoomCommandAccept),
			})

			d.DeleteRoomRequest(id)
			return nil
		}
	}

	return fmt.Errorf("room request with id %s not found", id)
}

func (d *Daemon) DeleteRoomRequest(id uuid.UUID) {
	for i := 0; i < len(d.data.Requests); i++ {
		if d.data.Requests[i].ID == id {
			d.data.Requests[len(d.data.Requests)-1], d.data.Requests[i] = d.data.Requests[i], d.data.Requests[len(d.data.Requests)-1]
			d.data.Requests = d.data.Requests[:len(d.data.Requests)-1]
			break
		}
	}
}
