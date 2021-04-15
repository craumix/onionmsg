package types

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"golang.org/x/net/proxy"
)

type Room struct {
	Self	*Identity
	Peers	[]*RemoteIdentity
	ID		uuid.UUID
}

func NewRoom(contactIdentities []*RemoteIdentity, proxy proxy.Dialer) (*Room, error) {
	s := NewIdentity()
	peers := make([]*RemoteIdentity, 0)
	id, _ := uuid.NewUUID()

	for _, c :=  range contactIdentities {
		con, err := proxy.Dial("tcp", c.URL() + ":10050")
		if err != nil {
			return nil, err
		}

		_, err = WriteCon(con, []byte(c.Fingerprint()))
		if err != nil {
			return nil, err
		}
		
		_, err = WriteCon(con, id[:])
		if err != nil {
			return nil, err
		}

		msg, err := ReadCon(con)
		if err != nil {
			return nil, err
		}

		sig, err := ReadCon(con)
		if err != nil {
			return nil, err
		}

		con.Close()

		if !c.Verify(append(msg, id[:]...), sig) {
			return nil, fmt.Errorf("Invalid signature from remote %s", c.URL())
		}

		r, err := NewRemoteIdentity(string(msg))
		if err != nil {
			return nil, err
		}

		log.Printf("Validated %s\n", c.URL())
		log.Printf("Conversiation ID %s\n", string(msg))

		peers = append(peers, r)
	}
	

	return &Room{
		Self: s,
		Peers: peers,
		ID: id,
	}, nil
}
