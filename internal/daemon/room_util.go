package daemon

import (
	"fmt"
	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
)

func (d *Daemon) initRooms() error {
	for _, room := range d.GetRooms() {
		err := d.serveConvIDService(room.Self)
		if err != nil {
			return err
		}
		room.RunMessageQueueForAllPeers()
	}

	return nil
}

func (d *Daemon) registerRoom(room *types.Room) error {
	err := d.serveConvIDService(room.Self)
	if err != nil {
		return err
	}

	d.AddRoom(room)

	log.WithField("room", room.ID.String()).Info("registered room")

	d.Notifier.NotifyNewRoom(room.Info())

	return nil
}

func (d *Daemon) serveConvIDService(i types.Identity) error {
	return d.Tor.RegisterService(*i.Priv, types.PubConvPort, d.loConvPort)
}

func (d *Daemon) deregisterRoom(id string) error {
	room, found := d.GetRoomByID(id)
	if !found {
		return nil
	}

	err := d.Tor.DeregisterService(*room.Self.Pub)
	if err != nil {
		return err
	}

	room.StopQueues()

	log.WithField("room", id).Info("degistered room")

	return nil
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

	room, err := types.NewRoom(d.ctx, d.ConnectionManager, ids...)
	if err != nil {
		return err
	}

	return d.registerRoom(room)
}

func (d *Daemon) AddNewPeerToRoom(roomID string, newPeerFingerprint string) error {
	room, found := d.GetRoomByID(roomID)
	if !found {
		return fmt.Errorf("no such room %s", roomID)
	}

	rID, err := types.NewIdentity(types.Remote, newPeerFingerprint)
	if err != nil {
		return err
	}

	return room.AddPeers(rID)
}

func (d *Daemon) DeregisterAndDeleteRoomByID(roomID string) error {
	err := d.deregisterRoom(roomID)
	if err != nil {
		return err
	}

	d.RemoveRoomByID(roomID)

	return nil
}

func (d *Daemon) GetInfoForAllRooms() []*types.RoomInfo {
	var roomInfos []*types.RoomInfo
	for _, r := range d.GetRooms() {
		roomInfos = append(roomInfos, r.Info())
	}

	return roomInfos
}

func (d *Daemon) SendMessageInRoom(roomID string, content types.MessageContent) error {
	room, found := d.GetRoomByID(roomID)
	if !found {
		return fmt.Errorf("no such room %s", roomID)
	}

	room.SendMessageToAllPeers(content)
	return nil
}

func (d *Daemon) ListMessagesInRoom(roomID string, count int) ([]types.Message, error) {
	room, found := d.GetRoomByID(roomID)
	if !found {
		return nil, fmt.Errorf("no such room %s", roomID)
	}

	if count > 0 && count < len(room.Messages) {
		return room.Messages[len(room.Messages)-count:], nil
	} else {
		return room.Messages, nil
	}
}

func (d *Daemon) GetRooms() []*types.Room {
	return d.data.Rooms
}

func (d *Daemon) AddRoom(room *types.Room) {
	d.data.Rooms = append(d.data.Rooms, room)
}

func (d *Daemon) RemoveRoom(toRemove *types.Room) {
	d.RemoveRoomByID(toRemove.ID.String())
}

func (d *Daemon) RemoveRoomByID(toRemove string) {
	for i, room := range d.GetRooms() {
		if room.ID.String() == toRemove {
			d.data.Rooms = append(d.data.Rooms[:i], d.data.Rooms[i+1:]...)
			return
		}
	}
}

func (d *Daemon) GetRoomByID(toFind string) (*types.Room, bool) {
	for _, room := range d.GetRooms() {
		if room.ID.String() == toFind {
			return room, true
		}
	}

	return nil, false
}

func (d *Daemon) GetRoomInfoByID(roomID string) (*types.RoomInfo, error) {
	room, found := d.GetRoomByID(roomID)
	if !found {
		return nil, fmt.Errorf("no such room %s", roomID)
	}

	return room.Info(), nil
}
