package daemon

import (
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
)

var (
	NewMessageCallback func(uuid.UUID, ...types.Message)
	NewRoomCallback    func(info *types.RoomInfo)
	ErrorCallback      func(error)
	NewRequestCallback func(*types.RoomRequest)
)

func notifyNewMessages(id uuid.UUID, msgs ...types.Message) {
	if NewMessageCallback != nil {
		go NewMessageCallback(id, msgs...)
	}
}

func notifyNewRoom(info *types.RoomInfo) {
	if NewMessageCallback != nil {
		go NewRoomCallback(info)
	}
}

func notifyError(err error) {
	if NewMessageCallback != nil {
		go ErrorCallback(err)
	}
}

func notifyNewRequest(req *types.RoomRequest) {
	if NewRequestCallback != nil {
		go NewRequestCallback(req)
	}
}
