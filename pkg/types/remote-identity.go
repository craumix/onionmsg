package types

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/google/uuid"
	"github.com/wybiral/torgo"
	"golang.org/x/net/proxy"
)

const (
	queueTimeout = time.Second * 15
	blocksize    = 4096
)

type RemoteIdentity struct {
	Pub   ed25519.PublicKey `json:"public_key"`
	Queue []*Message        `json:"queue"`

	sender           *Identity
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
	raw, err := base64.RawURLEncoding.DecodeString(fingerprint)
	if err != nil {
		return nil, err
	}

	return &RemoteIdentity{
		Pub: ed25519.PublicKey(raw),
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

func (i *RemoteIdentity) InitQueue(sender *Identity, dialer proxy.Dialer, conversationPort int, roomID uuid.UUID, terminate chan bool) {
	i.sender = sender
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

func (i *RemoteIdentity) sendMessage(msg *Message, dconn *sio.DataConn) error {
	sigSalt, err := dconn.ReadBytes()
	if err != nil {
		return err
	}

	meta, _ := json.Marshal(msg.Meta)

	_, err = dconn.WriteBytes(meta)
	if err != nil {
		return err
	}

	sig := i.sender.Sign(append(sigSalt, meta...))
	_, err = dconn.WriteBytes(sig)
	if err != nil {
		return err
	}
	dconn.Flush()

	resp, err := dconn.ReadInt()
	if err != nil {
		return err
	} else if resp != 0 {
		return fmt.Errorf("received invalid error response code %d", resp)
	}

	hash := sha256.New()
	hash.Write(sig)
	if msg.Meta.Type != MTYPE_BLOB {
		_, err = dconn.WriteBytes(msg.Content)
		if err != nil {
			return err
		}

		hash.Write(msg.Content)
	} else {
		id, _ := uuid.ParseBytes(msg.Content)
		stat, err := blobmngr.StatFromID(id)
		if err != nil {
			return err
		}

		blockcount := int(stat.Size() / blocksize)
		if stat.Size()%blocksize != 0 {
			blockcount++
		}

		_, err = dconn.WriteInt(blockcount)
		if err != nil {
			return err
		}

		file, err := blobmngr.FileFromID(id)
		if err != nil {
			return err
		}

		buf := make([]byte, blocksize)
		for i := 0; i < blockcount; i++ {
			n, err := file.Read(buf)
			if err != nil {
				return err
			}

			_, err = dconn.WriteBytes(buf[:n])
			if err != nil {
				return err
			}

			hash.Write(buf[:n])
		}
	}

	sig = i.sender.Sign(hash.Sum(nil))
	_, err = dconn.WriteBytes(sig)
	if err != nil {
		return err
	}

	dconn.Flush()

	resp, err = dconn.ReadInt()
	if err != nil {
		return err
	} else if resp != 0 {
		return fmt.Errorf("received invalid error response code %d", resp)
	}

	return nil
}
