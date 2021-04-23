package types

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Craumix/onionmsg/internal/sio"
	"github.com/google/uuid"
	"github.com/wybiral/torgo"
	"golang.org/x/net/proxy"
)

const (
	queueTimeout = time.Second * 15
)

type RemoteIdentity struct {
	Pub   ed25519.PublicKey `json:"public_key"`
	Queue []*Message        `json:"queue"`

	dialer           proxy.Dialer
	convPort         int
	roomID           uuid.UUID
	queueWasInit     bool
	lastQueueErr     error
	lastQueueRun     time.Time
	lastQueueRuntime int64
	queueTerminate   chan bool
}

func NewRemoteIdentity(fingerprint string) (*RemoteIdentity, error) {
	k, err := base64.RawURLEncoding.DecodeString(fingerprint)
	if err != nil {
		return nil, err
	}

	return &RemoteIdentity{
		Pub: ed25519.PublicKey(k),
	}, nil
}

func (i *RemoteIdentity) Verify(msg, sig []byte) bool {
	return ed25519.Verify(i.Pub, msg, sig)
}

func (i *RemoteIdentity) URL() string {
	return i.ServiceID() + ".onion"
}

func (i *RemoteIdentity) Fingerprint() string {
	return i.B64PubKey()
}

func (i *RemoteIdentity) B64PubKey() string {
	return base64.RawURLEncoding.EncodeToString(i.Pub)
}

func (i *RemoteIdentity) ServiceID() (id string) {
	id, _ = torgo.ServiceIDFromEd25519(i.Pub)
	return
}

func (i *RemoteIdentity) InitQueue(dialer proxy.Dialer, conversationPort int, roomID uuid.UUID, terminate chan bool) {
	i.dialer = dialer
	i.convPort = conversationPort
	i.roomID = roomID
	i.queueTerminate = terminate
	i.queueWasInit = true
}

func (i *RemoteIdentity) QueueMessage(msg *Message) {
	if i.queueWasInit {
		conn, err := i.dialer.Dial("tcp", i.URL()+":"+strconv.Itoa(i.convPort))
		if err == nil {
			dconn := sio.NewDataIO(conn)
			defer dconn.Close()

			dconn.WriteBytes(i.roomID[:])
			dconn.WriteInt(1)
			dconn.Flush()

			err = i.sendMessage(msg, dconn)
			if err == nil {
				return
			}
		}
	}

	i.Queue = append(i.Queue, msg)
}

func (i *RemoteIdentity) RunMessageQueue() error {
	if !i.queueWasInit {
		return fmt.Errorf("queue was not initialized")
	}

	for {
		if len(i.Queue) == 0 {
			time.Sleep(queueTimeout)
			continue
		}

		startTime := time.Now()
		conn, err := i.dialer.Dial("tcp", i.URL()+":"+strconv.Itoa(i.convPort))
		if err == nil {
			dconn := sio.NewDataIO(conn)
			defer dconn.Close()

			dconn.WriteBytes(i.roomID[:])
			dconn.WriteInt(len(i.Queue))
			dconn.Flush()
			for index, msg := range i.Queue {
				err = i.sendMessage(msg, dconn)
				if err != nil {
					log.Println(err.Error())
					break
				}

				copy(i.Queue[index:], i.Queue[index+1:]) // Shift a[i+1:] left one index.
				i.Queue[len(i.Queue)-1] = nil            // Erase last element (write zero value).
				i.Queue = i.Queue[:len(i.Queue)-1]       // Truncate slice.
			}
		}

		i.lastQueueRun = startTime
		i.lastQueueRuntime = time.Since(startTime).Nanoseconds()
		i.lastQueueErr = err

		select {
            case <-i.queueTerminate:
                return fmt.Errorf("queue terminated")
			default:
        }

		time.Sleep(queueTimeout)

		select {
            case <-i.queueTerminate:
                return fmt.Errorf("queue terminated")
			default:
        }
	}
}

func (i *RemoteIdentity) sendMessage(msg *Message, dconn *sio.DataConn) (err error) {
	raw, _ := json.Marshal(msg)

	_, err = dconn.WriteBytes(raw)
	if err != nil {
		return
	}
	dconn.Flush()

	state, err := dconn.ReadBytes()
	if err != nil {
		return
	}

	if state[0] != 0x00 {
		err = fmt.Errorf("received invalid state for message %d", state[0])
	}

	return
}
