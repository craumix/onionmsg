package types

import (
	"context"
	"fmt"
	"github.com/craumix/onionmsg/pkg/sio/connection"
	"log"
	"time"
)

const (
	queueTimeout = time.Second * 15
)

type MessagingPeer struct {
	RIdentity RemoteIdentity `json:"identity"`
	MQueue    []Message      `json:"queue"`

	ctx  context.Context
	stop context.CancelFunc

	Room *Room `json:"-"`
}

func NewMessagingPeer(rid RemoteIdentity) *MessagingPeer {
	return &MessagingPeer{
		RIdentity: rid,
	}
}

// RunMessageQueue creates a cancellable context for the MessagingPeer
// and starts a loop that will try to send queued messages every so often.
func (mp *MessagingPeer) RunMessageQueue(ctx context.Context, room *Room) {
	mp.Room = room

	mp.ctx, mp.stop = context.WithCancel(ctx)

	for {
		select {
		case <-mp.ctx.Done():
			log.Printf("Queue with %s in %s terminated!\n", mp.RIdentity.Fingerprint(), room.ID.String())
			return
		default:
			if len(mp.MQueue) == 0 {
				break
			}

			c, err := mp.SendMessages(mp.MQueue...)
			if err != nil {
				//TODO Uncomment
				//log.Println(err)
			} else {
				mp.MQueue = mp.MQueue[c:]
			}
		}
		select {
		case <-mp.ctx.Done(): //context cancelled
		case <-time.After(queueTimeout): //timeout
		}
	}
}

// QueueMessage tries to send the message right away and if that fails the message will be queued
func (mp *MessagingPeer) QueueMessage(msg Message) {
	_, err := mp.SendMessages(msg)
	if err != nil {
		mp.MQueue = append(mp.MQueue, msg)
	}
}

func (mp *MessagingPeer) SendMessages(msgs ...Message) (int, error) {
	if mp.Room == nil {
		return 0, fmt.Errorf("Room not set")
	}

	dataConn, err := connection.GetConnFunc("tcp", mp.getConvURL())
	if err != nil {
		return 0, err
	}

	defer dataConn.Close()

	dataConn.WriteBytes(mp.Room.ID[:])
	dataConn.WriteInt(len(msgs))
	dataConn.Flush()

	for index, msg := range msgs {
		err = SendMessage(&dataConn, mp.Room.Self, msg)
		if err != nil {
			dataConn.Close()
			return index, err
		}
	}

	return len(msgs), nil
}

// TerminateMessageQueue cancels the context for this MessagingPeer
func (mp *MessagingPeer) TerminateMessageQueue() {
	mp.stop()
}

func (mp *MessagingPeer) getConvURL() string {
	return fmt.Sprintf("%s:%d", mp.RIdentity.URL(), PubConvPort)
}
