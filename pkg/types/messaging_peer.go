package types

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/google/uuid"
)

const (
	queueTimeout = time.Second * 15
	// 512K
	blocksize = 1 << 19
)

type MessagingPeer struct {
	RIdentity RemoteIdentity `json:"identity"`
	MQueue    []Message      `json:"queue"`

	ctx  context.Context
	stop context.CancelFunc

	room *Room
}

func NewMessagingPeer(rid RemoteIdentity) *MessagingPeer {
	return &MessagingPeer{
		RIdentity: rid,
	}
}

// RunMessageQueue creates a cancellable context for the MessagingPeer
// and starts a loop that will try to send queued messages every so often.
func (mp *MessagingPeer) RunMessageQueue(ctx context.Context, room *Room) error {
	mp.room = room

	mp.ctx, mp.stop = context.WithCancel(ctx)

	for {
		select {
		case <-mp.ctx.Done():
			log.Printf("Queue with %s in %s terminated!\n", mp.RIdentity.Fingerprint(), room.ID.String())
			mp.stop()
			return nil
		default:
			if len(mp.MQueue) == 0 {
				continue
			}

			c, err := mp.transferMessages(mp.MQueue...)
			if err != nil {
				log.Println(err)
			} else {
				mp.MQueue = mp.MQueue[c:]
			}
		}
		time.Sleep(queueTimeout)
	}
}

// QueueMessage tries to send the message right away and if that fails the message will be queued
func (mp *MessagingPeer) QueueMessage(msg Message) {
	_, err := mp.transferMessages(msg)
	if err != nil {
		mp.MQueue = append(mp.MQueue, msg)
	}
}

func (mp *MessagingPeer) transferMessages(msgs ...Message) (int, error) {
	if mp.room == nil {
		return 0, fmt.Errorf("room not set")
	}

	dataConn, err := sio.DialDataConn("tcp", mp.getConvURL())
	if err != nil {
		return 0, err
	}
	defer dataConn.Close()

	dataConn.WriteBytes(mp.room.ID[:])
	dataConn.WriteInt(len(msgs))
	dataConn.Flush()

	for index, msg := range msgs {
		err = mp.sendMessage(msg, dataConn)
		if err != nil {
			dataConn.Close()
			return index, err
		}
	}

	return len(msgs), nil
}

func (mp *MessagingPeer) getConvURL() string {
	return fmt.Sprintf("%s:%d", mp.RIdentity.URL(), PubConvPort)
}

func (mp *MessagingPeer) sendMessage(msg Message, dataConn *sio.DataConn) error {
	sigSalt, err := dataConn.ReadBytes()
	if err != nil {
		return err
	}

	meta, _ := json.Marshal(msg.Meta)
	_, err = mp.sendDataWithSig(dataConn, meta, sigSalt)
	if err != nil {
		return nil
	}

	if msg.Meta.Type != MessageTypeBlob {
		_, err = mp.sendDataWithSig(dataConn, msg.Content, sigSalt)
		if err != nil {
			return nil
		}
	} else {
		id, err := uuid.FromBytes(msg.Content)
		if err != nil {
			return err
		}

		stat, err := blobmngr.StatFromID(id)
		if err != nil {
			return err
		}

		blockCount := int(stat.Size() / blocksize)
		if stat.Size()%blocksize != 0 {
			blockCount++
		}

		_, err = dataConn.WriteInt(blockCount)
		if err != nil {
			return err
		}

		file, err := blobmngr.FileFromID(id)
		if err != nil {
			return err
		}
		defer file.Close()

		buf := make([]byte, blocksize)
		for c := 0; c < blockCount; c++ {
			n, err := file.Read(buf)
			if err != nil {
				return err
			}

			_, err = mp.sendDataWithSig(dataConn, buf[:n], sigSalt)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (mp *MessagingPeer) sendDataWithSig(dataConn *sio.DataConn, data, sigSalt []byte) (int, error) {
	n, err := dataConn.WriteBytes(data)
	if err != nil {
		return 0, err
	}
	m, err := dataConn.WriteBytes(mp.room.Self.Sign(append(sigSalt, data...)))
	if err != nil {
		return n, err
	}
	dataConn.Flush()

	resp, err := dataConn.ReadString()
	if err != nil {
		return m + n, err
	} else if resp != "ok" {
		return m + n, fmt.Errorf("received response \"%s\" for msg meta", resp)
	}

	return m + n, nil
}

// TerminateMessageQueue cancels the context for this MessagingPeer
func (mp *MessagingPeer) TerminateMessageQueue() {
	mp.stop()
}
