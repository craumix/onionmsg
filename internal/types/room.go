package types

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/proxy"

	"github.com/Craumix/tormsg/internal/sio"
)

type Room struct {
	Self		*Identity			`json:"self"`
	Peers		[]*RemoteIdentity	`json:"peers"`
	ID			uuid.UUID			`json:"uuid"`
	Messages	[]*Message			`json:"messages"`
}

func NewRoom(contactIdentities []*RemoteIdentity, dialer proxy.Dialer, contactPort, conversationPort int) (*Room, error) {
	s := NewIdentity()
	peers := make([]*RemoteIdentity, 0)
	id, _ := uuid.NewUUID()

	for _, c :=  range contactIdentities {
		conn, err := dialer.Dial("tcp", c.URL() + ":" + strconv.Itoa(contactPort))
		if err != nil {
			return nil, err
		}

		dconn := sio.NewDataIO(conn)

		_, err = dconn.WriteString(c.Fingerprint())
		if err != nil {
			return nil, err
		}

		_, err = dconn.WriteString(s.Fingerprint())
		if err != nil {
			return nil, err
		}
		
		_, err = dconn.WriteBytes(id[:])
		if err != nil {
			return nil, err
		}

		dconn.Flush()

		remoteConv, err := dconn.ReadString()
		if err != nil {
			return nil, err
		}

		sig, err := dconn.ReadBytes()
		if err != nil {
			return nil, err
		}

		dconn.Close()

		if !c.Verify(append([]byte(remoteConv), id[:]...), sig) {
			return nil, fmt.Errorf("invalid signature from remote %s", c.URL())
		}

		r, err := NewRemoteIdentity(remoteConv)
		if err != nil {
			return nil, err
		}

		log.Printf("Validated %s\n", c.URL())
		log.Printf("Conversiation ID %s\n", remoteConv)

		peers = append(peers, r)
	}
	
	room := &Room{
		Self: s,
		Peers: peers,
		ID: id,
		Messages: make([]*Message, 0),
	}

	for _, peer := range peers {
		room.SendMessage(MTYPE_CMD, []byte("join " + peer.Fingerprint()))
	}

	return room, nil
}

func (r *Room) SendMessage(mtype byte, content []byte) {
	msg := &Message{
		Sender: r.Self.Fingerprint(),
		Time: time.Now(),
		Type: mtype,
		Content: content,
	}
	msg.Sign(r.Self.Key)

	r.Messages = append(r.Messages, msg)

	for _, peer := range r.Peers {
		peer.Queue = append(peer.Queue, msg)
	}
}

func (r *Room) RunRemoteMessageQueues(dialer proxy.Dialer, conversationPort int) {
	for _, peer := range r.Peers {
		go peer.RunMessageQueue(dialer, conversationPort)
	}
}
