package tests

import (
	"context"
	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
	"testing"
	"time"
)

var (
	peer    *types.MessagingPeer
	message types.Message
	room    types.Room
)

func setup() {
	connection.GetConnFunc = GetMockedConnWrapper

	MockedConn = &MockConnWrapper{}

	identity, _ := types.NewRemoteIdentity("Test")
	peer = types.NewMessagingPeer(identity)

	message = types.Message{
		Meta: types.MessageMeta{
			Sender: "test",
			Time:   time.Time{},
			Type:   "mtype.text",
		},
		Content:     []byte("this is a test"),
		ContentMeta: nil,
	}

	room = types.Room{
		Self:     types.NewIdentity(),
		Peers:    nil,
		ID:       uuid.New(),
		Name:     "",
		Messages: nil,
	}

	room.SetContext(context.TODO())
}

func TestQueueMessage(t *testing.T) {
	setup()
	if len(peer.MQueue) != 0 {
		t.Error("Peer doesn't start with an empty Message queue")
	}

	peer.QueueMessage(message)

	if len(peer.MQueue) != 1 {
		t.Error("Message not queued")
	}
}

func TestRunMessageQueue(t *testing.T) {
	setup()
	peer.QueueMessage(message)
	go peer.RunMessageQueue(room.Ctx, &room)

	time.Sleep(time.Second)

	if len(peer.MQueue) != 0 {
		t.Error("Message was not transferred!")
	}
}

func TestRunMessageQueueContextCancelled(t *testing.T) {
	setup()
	room.StopQueues()
	peer.QueueMessage(message)
	peer.RunMessageQueue(room.Ctx, &room)

	if len(peer.MQueue) != 1 {
		t.Error("Message transferred while queue is cancelled!")
	}
}

func TestTransferMessage(t *testing.T) {
	setup()
	room.StopQueues()
	peer.RunMessageQueue(room.Ctx, &room)
	peer.TransferMessages(message)

	if !sameArray(MockedConn.WriteBytesInput[0], room.ID[:]) {
		t.Error("Wrong room ID!")
	}

	if MockedConn.WriteIntInput[0] != 1 {
		t.Error("Wrong amount of messages!")
	}

	if !MockedConn.FlushCalled {
		t.Error("Connection was not flushed!")
	}

	if !MockedConn.CloseCalled {
		t.Error("Connection was not closed!")
	}
}

func sameArray(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		// println("%b\n%b", a[i], b[i])
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
