package daemon

import (
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
)

var (
	NewMessageCallback func(uuid.UUID, types.Message)
	NewRoomCallback func(uuid.UUID)
	ErrorCallback func(error)
)

func notifyNewMessage(id uuid.UUID, msg types.Message) {
	if NewMessageCallback != nil {
		go NewMessageCallback(id, msg)
	}
}

func notifyNewRoom(id uuid.UUID) {
	if NewMessageCallback != nil {
		go NewRoomCallback(id)
	}
}

func notifyError(err error) {
	if NewMessageCallback != nil {
		go ErrorCallback(err)
	}
}