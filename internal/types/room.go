package types

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"golang.org/x/net/proxy"
)

type Room struct {
	Self		*Identity			`json:"self"`
	Peers		[]*RemoteIdentity	`json:"peers"`
	ID			uuid.UUID			`json:"uuid"`
	Messages	[]*Message			`json:"messages"`
}

func NewRoom(contactIdentities []*RemoteIdentity, proxy proxy.Dialer) (*Room, error) {
	s := NewIdentity()
	peers := make([]*RemoteIdentity, 0)
	id, _ := uuid.NewUUID()

	for _, c :=  range contactIdentities {
		conn, err := proxy.Dial("tcp", c.URL() + ":10050")
		if err != nil {
			return nil, err
		}

		dconn := NewDataIO(conn)

		_, err = dconn.WriteString(c.Fingerprint())
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
			return nil, fmt.Errorf("Invalid signature from remote %s", c.URL())
		}

		r, err := NewRemoteIdentity(remoteConv)
		if err != nil {
			return nil, err
		}

		log.Printf("Validated %s\n", c.URL())
		log.Printf("Conversiation ID %s\n", remoteConv)

		peers = append(peers, r)
	}
	

	return &Room{
		Self: s,
		Peers: peers,
		ID: id,
	}, nil
}
