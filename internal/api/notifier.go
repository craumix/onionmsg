package api

import (
	"log"

	"github.com/craumix/onionmsg/internal/daemon"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type NotificationType string

const (
	NotificationTypeNewMessage = "NewMessage"
)

var (
	observerList []*websocket.Conn
)

func init() {
	daemon.MessageNotificationListener = NotifyNewMessage
}

func NotifyNewMessage(id uuid.UUID, msg types.Message) {
	n := struct {
		RoomID  uuid.UUID     `json:"uuid"`
		Message types.Message `json:"message"`
	}{
		id,
		msg,
	}

	NotifyObservers(NotificationTypeNewMessage, n)
}

func NotifyObservers(ntype NotificationType, msg interface{}) {
	notification := struct {
		Type NotificationType `json:"type"`
		Data interface{}      `json:"data"`
	}{
		ntype,
		msg,
	}

	go func() {
		for _, c := range observerList {
			err := c.WriteJSON(notification)
			if err != nil {
				//TODO remove dead sockets
				log.Print(err)
				c.Close()
			}
		}
	}()
}
