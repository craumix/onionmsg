package types

import (
	"github.com/google/uuid"
	"io"
)

type NotificationType string

const (
	NotificationTypeNewMessage NotificationType = "NewMessage"
	NotificationTypeNewRoom    NotificationType = "NewRoom"
	NotificationTypeError      NotificationType = "Error"
	NotificationTypeNewRequest NotificationType = "NewRequest"
)

type Observer interface {
	io.Closer
	WriteJSON(v interface{}) error
}

type Notifier struct {
	observers []Observer
}

type Notification struct {
	Type NotificationType `json:"type"`
	Data interface{}      `json:"data"`
}

func (n *Notifier) AddObserver(newObserver Observer) {
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

	n.notifyObservers(Notification{NotificationTypeNewMessage, data})
}

func (n *Notifier) NotifyNewRoom(info *RoomInfo) {
	n.notifyObservers(Notification{NotificationTypeNewRoom, info})
}

func (n *Notifier) NotifyError(err error) {
	n.notifyObservers(Notification{NotificationTypeError, err.Error()})
}

func (n *Notifier) NotifyNewRequest(req *RoomRequest) {
	n.notifyObservers(Notification{NotificationTypeNewRequest, req})
}

func (n *Notifier) notifyObservers(notification Notification) {
	for _, conn := range n.observers {
		err := conn.WriteJSON(notification)
		if err != nil {
			//TODO remove dead sockets
			conn.Close()
		}
	}
}
