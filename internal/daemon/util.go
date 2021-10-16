package daemon

import (
	"fmt"
	"github.com/craumix/onionmsg/internal/types"
	"strings"
)

type TorStringWriter struct {
	OnWrite func(string)
}

func (w TorStringWriter) Write(p []byte) (int, error) {
	if w.OnWrite != nil {
		lines := strings.Split(string(p), "\n")
		lines = lines[:len(lines)-1]
		for _, v := range lines {
			w.OnWrite(v)
		}
	}
	return len(p), nil
}

func (d *Daemon) AcceptRoomRequest(toAccept string) error {
	request, found := d.GetRoomRequestByID(toAccept)
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

	d.RemoveContactIDByFingerprint(toAccept)

	return nil
}

func (d *Daemon) GetRoomRequests() []*types.RoomRequest {
	return d.data.Requests
}

func (d *Daemon) AddRoomRequest(request *types.RoomRequest) {
	d.data.Requests = append(d.data.Requests, request)
}

func (d *Daemon) RemoveRoomRequest(toRemove *types.RoomRequest) {
	d.RemoveRoomRequestByID(toRemove.ID.String())
}

func (d *Daemon) RemoveRoomRequestByID(toRemove string) {
	for i, request := range d.GetRoomRequests() {
		if request.ID.String() == toRemove {
			d.data.Requests = append(d.data.Requests[:i], d.data.Requests[i+1:]...)
			return
		}
	}
}

func (d *Daemon) GetRoomRequestByID(toFind string) (*types.RoomRequest, bool) {
	for _, request := range d.GetRoomRequests() {
		if request.ID.String() == toFind {
			return request, true
		}
	}

	return nil, false
}
