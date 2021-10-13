package types

import (
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

type Notifier struct {
	observers []*websocket.Conn
}

func (n *Notifier) AddObserver(newObserver *websocket.Conn) {
	n.observers = append(n.observers, newObserver)
}

func (n *Notifier) NotifyNewMessage(id uuid.UUID, msg ...Message) {
	data := struct {
		RoomID  uuid.UUID `json:"uuid"`
		Message []Message `json:"messages"`
	}{
		id,
		msg,
	}

	n.notifyObservers(NotificationTypeNewMessage, data)
}

func (n *Notifier) NotifyNewRoom(info *RoomInfo) {
	n.notifyObservers(NotificationTypeNewRoom, info)
}

func (n *Notifier) NotifyError(err error) {
	n.notifyObservers(NotificationTypeError, err.Error())
}

func (n *Notifier) NotifyNewRequest(req *RoomRequest) {
	n.notifyObservers(NotificationTypeNewRequest, req)
}

func (n *Notifier) notifyObservers(ntype NotificationType, msg interface{}) {
	notification := struct {
		Type NotificationType `json:"type"`
		Data interface{}      `json:"data"`
	}{
		ntype,
		msg,
	}

	for _, conn := range n.observers {
		err := conn.WriteJSON(notification)
		if err != nil {
			//TODO remove dead sockets
			conn.Close()
		}
	}
}
