package types

import (
	"fmt"
	"log"
	"strconv"

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
		go r.RunMessageQueue(dialer, conversationPort)

		log.Printf("Validated %s\n", c.URL())
		log.Printf("Conversiation ID %s\n", remoteConv)

		peers = append(peers, r)
	}
	

	return &Room{
		Self: s,
		Peers: peers,
		ID: id,
		Messages: make([]*Message, 0),
	}, nil
}
