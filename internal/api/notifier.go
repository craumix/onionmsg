package api

import (
	"log"

	"github.com/craumix/onionmsg/internal/daemon"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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

	NotifyObservers(n)
}

func NotifyObservers(msg interface{}) {
	go func() {
		for _, c := range observerList {
			err := c.WriteJSON(msg)
			if err != nil {
				//TODO remove dead sockets
				log.Print(err.Error())
				c.Close()
			}
		}
	}()
}
