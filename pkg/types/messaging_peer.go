package types

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/google/uuid"
	"golang.org/x/net/proxy"
)

const (
	queueTimeout = time.Second * 15
	//512K
	blocksize = 1 << 19
)

type MessagingPeer struct {
	MQueue    []*Message      `json:"queue"`
	RIdentity *RemoteIdentity `json:"identity"`

	room   *Room
	dialer proxy.Dialer
}

func NewMessagingPeer(rid *RemoteIdentity) *MessagingPeer {
	return &MessagingPeer{
		RIdentity: rid,
	}
}

func (mp *MessagingPeer) RunMessageQueue(dialer proxy.Dialer, room *Room) error {
	mp.dialer = dialer
	mp.room = room

	for {
		if len(mp.MQueue) == 0 {
			time.Sleep(queueTimeout)
			continue
		}

		c, err := mp.transferMessages(mp.MQueue...)
		if err != nil {
			log.Println(err)
		} else {
			mp.MQueue = mp.MQueue[c:]
		}

		select {
		case <-room.queueTerminate:
			return fmt.Errorf("queue terminated")
		default:
		}

		time.Sleep(queueTimeout)

		select {
		case <-room.queueTerminate:
			return fmt.Errorf("queue terminated")
		default:
		}
	}
}

func (mp *MessagingPeer) QueueMessage(msg *Message) {
	_, err := mp.transferMessages(msg)
	if err != nil {
		mp.MQueue = append(mp.MQueue, msg)
	}
}

func (mp *MessagingPeer) transferMessages(msgs ...*Message) (int, error) {
	if mp.dialer == nil || mp.room == nil {
		return 0, fmt.Errorf("dialer or room not set")
	}

	conn, err := mp.dialer.Dial("tcp", mp.getConvURL())
	if err != nil {
		return 0, err
	}

	dconn := sio.NewDataIO(conn)
	defer dconn.Close()

	dconn.WriteBytes(mp.room.ID[:])
	dconn.WriteInt(len(msgs))
	dconn.Flush()

	for index, msg := range msgs {
		err = mp.sendMessage(msg, dconn)
		if err != nil {
			dconn.Close()
			return index, err
		}
	}

	return len(msgs), nil
}

func (mp *MessagingPeer) getConvURL() string {
	return fmt.Sprintf("%s:%d", mp.RIdentity.URL(), PubConvPort)
}

func (mp *MessagingPeer) sendMessage(msg *Message, dconn *sio.DataConn) error {
	sigSalt, err := dconn.ReadBytes()
	if err != nil {
		return err
	}

	meta, _ := json.Marshal(msg.Meta)
	_, err = mp.sendDataWithSig(dconn, meta, sigSalt)
	if err != nil {
		return nil
	}

	if msg.Meta.Type != MTYPE_BLOB {
		_, err = mp.sendDataWithSig(dconn, msg.Content, sigSalt)
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
		defer file.Close()

		buf := make([]byte, blocksize)
		for c := 0; c < blockcount; c++ {
			n, err := file.Read(buf)
			if err != nil {
				return err
			}

			_, err = mp.sendDataWithSig(dconn, buf[:n], sigSalt)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (mp *MessagingPeer) sendDataWithSig(dconn *sio.DataConn, data, sigSalt []byte) (int, error) {
	n, err := dconn.WriteBytes(data)
	if err != nil {
		return 0, err
	}
	m, err := dconn.WriteBytes(mp.room.Self.Sign(append(sigSalt, data...)))
	if err != nil {
		return n, err
	}
	dconn.Flush()

	resp, err := dconn.ReadString()
	if err != nil {
		return m + n, err
	} else if resp != "ok" {
		return m + n, fmt.Errorf("received response \"%s\" for msg meta", resp)
	}

	return m + n, nil
}
