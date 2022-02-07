package daemon

import (
	"fmt"
	"github.com/craumix/onionmsg/internal/types"
	"github.com/google/uuid"
)

func (d *Daemon) AcceptRoomRequest(toAccept uuid.UUID) error {
	request, found := d.getRoomRequest(toAccept)
	if !found {
		return fmt.Errorf("room request with toAccept %s not found", toAccept)
	}

	request.Room.SetContext(d.ctx)
	request.Room.SetConnectionManager(d.ConnectionManager)
	request.Room.SetCommandHandler(types.GetDefaultCommandHandler())

	err := d.registerRoom(&request.Room)
	if err != nil {
		return err
	}

	request.Room.RunMessageQueueForAllPeers()

	request.Room.SendMessageToAllPeers(types.MessageContent{
		Type: types.ContentTypeCmd,
		Data: types.ConstructCommand(nil, types.RoomCommandAccept),
	})

	d.removeContactID(types.Fingerprint(request.ViaFingerprint)) // FIXME Something is wrong here

	return nil
}

func (d *Daemon) DeleteRoomRequest(toRemove uuid.UUID) {
	for i, request := range d.GetRoomRequests() {
		if request.ID == toRemove {
			d.data.Requests = append(d.data.Requests[:i], d.data.Requests[i+1:]...)
			return
		}
	}
}

func (d *Daemon) GetRoomRequests() []*types.RoomRequest {
	return d.data.Requests
}

func (d *Daemon) getRoomRequest(toFind uuid.UUID) (*types.RoomRequest, bool) {
	for _, request := range d.GetRoomRequests() {
		if request.ID == toFind {
			return request, true
		}
	}

	return nil, false
}

func (d *Daemon) addRoomRequest(request *types.RoomRequest) {
	d.data.Requests = append(d.data.Requests, request)
}
