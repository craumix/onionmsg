package daemon

import (
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
)

func (d *Daemon) initRooms() error {
	for _, room := range d.GetRooms() {
		err := d.serveConvIDService(*room.Self)
		if err != nil {
			return err
		}
		room.RunMessageQueueForAllPeers()
	}

	return nil
}

func (d *Daemon) registerRoom(room *types.Room) error {
	err := d.serveConvIDService(*room.Self)
	if err != nil {
		return err
	}

	d.AddRoom(room)

	log.WithField("room", room.ID.String()).Info("registered room")

	d.Notifier.NotifyNewRoom(room.Info())

	return nil
}

func (d *Daemon) serveConvIDService(i types.SelfIdentity) error {
	return d.Tor.RegisterService(i.Priv, d.loConvPort, d.loConvPort)
}

func (d *Daemon) deregisterRoom(id uuid.UUID) error {
	room, found := d.GetRoomByID(id)
	if !found {
		return nil
	}

	err := d.Tor.DeregisterService(room.Self.Pub)
	if err != nil {
		return err
	}

	room.StopQueues()

	log.WithField("room", id).Info("degistered room")

	return nil
}

// Maybe this should be run in a goroutine
func (d *Daemon) CreateRoom(fingerprints []string) error {
	var ids []types.RemoteIdentity
	for _, fingerprint := range fingerprints {
		id, err := types.NewRemoteIdentity(types.Fingerprint(fingerprint))
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	room, err := types.NewRoom(d.ctx, d.ConnectionManager, types.GetDefaultCommandHandler(), ids...)
	if err != nil {
		return err
	}

	return d.registerRoom(room)
}

func (d *Daemon) AddNewPeerToRoom(roomID uuid.UUID, newPeerFingerprint types.Fingerprint) error {
	room, found := d.GetRoomByID(roomID)
	if !found {
		return fmt.Errorf("no such room %s", roomID)
	}

	rID, err := types.NewRemoteIdentity(newPeerFingerprint)
	if err != nil {
		return err
	}

	return room.AddPeers(rID)
}

func (d *Daemon) DeregisterAndDeleteRoom(roomID uuid.UUID) error {
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

func (d *Daemon) SendMessageInRoom(roomID uuid.UUID, content types.MessageContent) error {
	room, found := d.GetRoomByID(roomID)
	if !found {
		return fmt.Errorf("no such room %s", roomID)
	}

	room.SendMessageToAllPeers(content)
	return nil
}

func (d *Daemon) ListMessagesInRoom(roomID uuid.UUID, count int) ([]types.Message, error) {
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
	d.RemoveRoomByID(toRemove.ID)
}

func (d *Daemon) RemoveRoomByID(toRemove uuid.UUID) {
	for i, room := range d.GetRooms() {
		if room.ID == toRemove {
			d.data.Rooms = append(d.data.Rooms[:i], d.data.Rooms[i+1:]...)
			return
		}
	}
}

func (d *Daemon) GetRoomByID(toFind uuid.UUID) (*types.Room, bool) {
	for _, room := range d.GetRooms() {
		if room.ID == toFind {
			return room, true
		}
	}

	return nil, false
}

func (d *Daemon) GetRoomInfo(roomID uuid.UUID) (*types.RoomInfo, error) {
	room, found := d.GetRoomByID(roomID)
	if !found {
		return nil, fmt.Errorf("no such room %s", roomID)
	}

	return room.Info(), nil
}
