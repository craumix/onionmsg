package daemon

import (
	"github.com/craumix/onionmsg/internal/types"
	"github.com/google/uuid"
)

var (
	NewMessageHook func(uuid.UUID, ...types.Message)
	NewRoomHook    func(info *types.RoomInfo)
	ErrorHook      func(error)
	NewRequestHook func(*types.RoomRequest)
)

func notifyNewMessages(id uuid.UUID, msgs ...types.Message) {
	if NewMessageHook != nil {
		go NewMessageHook(id, msgs...)
	}
}

func notifyNewRoom(info *types.RoomInfo) {
	if NewMessageHook != nil {
		go NewRoomHook(info)
	}
}

func notifyError(err error) {
	if NewMessageHook != nil {
		go ErrorHook(err)
	}
}

func notifyNewRequest(req *types.RoomRequest) {
	if NewRequestHook != nil {
		go NewRequestHook(req)
	}
}
