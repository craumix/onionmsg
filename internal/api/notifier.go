package api

import (
	"github.com/craumix/onionmsg/internal/daemon"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type NotificationType string

const (
	NotificationTypeNewMessage = "NewMessage"
	NotificationTypeNewRoom    = "NewRoom"
	NotificationTypeError      = "Error"
	NotificationTypeNewRequest = "NewRequest"
)

var (
	observerList []*websocket.Conn
)

func init() {
	registerCallbacks()
}

func registerCallbacks() {
	daemon.NewMessageCallback = NotifyNewMessage
	daemon.NewRoomCallback = NotifyNewRoom
	daemon.ErrorCallback = NotifyError
	daemon.NewRequestCallback = NotifyNewRequest
}

func NotifyNewMessage(id uuid.UUID, msg ...types.Message) {
	n := struct {
		RoomID  uuid.UUID       `json:"uuid"`
		Message []types.Message `json:"messages"`
	}{
		id,
		msg,
	}

	NotifyObservers(NotificationTypeNewMessage, n)
}

func NotifyNewRoom(info *types.RoomInfo) {
	NotifyObservers(NotificationTypeNewRoom, info)
}

func NotifyError(err error) {
	NotifyObservers(NotificationTypeError, err.Error())
}

func NotifyNewRequest(req *types.RoomRequest) {
	NotifyObservers(NotificationTypeNewRequest, req)
}

func NotifyObservers(ntype NotificationType, msg interface{}) {
	notification := struct {
		Type NotificationType `json:"type"`
		Data interface{}      `json:"data"`
	}{
		ntype,
		msg,
	}

	for _, c := range observerList {
		err := c.WriteJSON(notification)
		if err != nil {
			//TODO remove dead sockets
			//log.Print(err)
			c.Close()
		}
	}
}
