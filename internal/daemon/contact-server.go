package daemon

import (
	"log"
	"net"
	"strconv"

	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
)

func startContactServer() error {
	server, err := net.Listen("tcp", "localhost:"+strconv.Itoa(contactPort))
	if err != nil {
		return err
	}
	defer server.Close()

	for {
		c, err := server.Accept()
		if err != nil {
			log.Println(err)
		}

		go func() {
			dconn := sio.NewDataIO(c)
			defer dconn.Close()

			contactFingerprint, err := dconn.ReadString()
			if err != nil {
				log.Println(err.Error())
				return
			}

			if data.ContactIdentities[contactFingerprint] == nil {
				log.Printf("Contact id %s unknown\n", contactFingerprint)
				return
			}

			remoteFingerprint, err := dconn.ReadString()
			if err != nil {
				log.Println(err.Error())
				return
			}
			remoteID, _ := types.NewRemoteIdentity(remoteFingerprint)
			if err != nil {
				log.Println(err.Error())
				return
			}

			msg, err := dconn.ReadBytes()
			if err != nil {
				log.Println(err.Error())
				return
			}
			id, _ := uuid.FromBytes(msg)

			convID := types.NewIdentity()
			_, err = dconn.WriteString(convID.Fingerprint())
			if err != nil {
				log.Println(err.Error())
				return
			}

			_, err = dconn.WriteBytes(data.ContactIdentities[contactFingerprint].Sign(append([]byte(convID.Fingerprint()), id[:]...)))
			if err != nil {
				log.Println(err.Error())
				return
			}

			dconn.Flush()

			room := &types.Room{
				Self:     convID,
				Peers:    []*types.RemoteIdentity{remoteID},
				ID:       id,
				Messages: make([]*types.Message, 0),
			}
			err = registerRoom(room)
			if err != nil {
				log.Println()
			}

			//Kinda breaks interactive
			//log.Printf("Exchange succesfull uuid %s sent id %s", id, convID.Fingerprint())
		}()
	}
}
