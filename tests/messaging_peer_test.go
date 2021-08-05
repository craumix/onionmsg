package tests

import (
	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/craumix/onionmsg/pkg/types"
	"os"
	"testing"
	"time"
)

var (
	peer    *types.MessagingPeer
	message types.Message
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func setup() {
	connection.GetConnFunc = GetMockedConnWrapper

	MockedConn = MockConnWrapper{}

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
}

func TestQueueMessage(t *testing.T) {
	MockedConn.ReadStringOutputString = "success"
	if len(peer.MQueue) != 0 {
		t.Error("Peer doesn't start with an empty Message queue")
	}

	peer.QueueMessage(message)

	if len(peer.MQueue) != 1 {
		t.Error("Message not queued")
	}
}
