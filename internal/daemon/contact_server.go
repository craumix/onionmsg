package daemon

import (
	"log"
	"net"
	"strconv"

	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/craumix/onionmsg/pkg/types"
)

func startContactServer() error {
	server, err := net.Listen("tcp", "localhost:"+strconv.Itoa(loContPort))
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

			req := &types.ContactRequest{}
			err = dconn.ReadStruct(req)
			if err != nil {
				log.Println(err.Error())
				return
			}

			cont, ok := GetContactID(req.RemoteFP)
			if !ok {
				log.Printf("Contact id %s unknown\n", req.RemoteFP)
				return
			}

			remoteID, _ := types.NewRemoteIdentity(req.LocalFP)

			convID := types.NewIdentity()

			resp := &types.ContactResponse{
				ConvFP: convID.Fingerprint(),
				Sig:    cont.Sign(append([]byte(convID.Fingerprint()), req.ID[:]...)),
			}

			_, err = dconn.WriteStruct(resp)
			if err != nil {
				log.Println(err.Error())
				return
			}

			dconn.Flush()

			room := &types.Room{
				Self:  convID,
				Peers: []*types.MessagingPeer{types.NewMessagingPeer(remoteID)},
				ID:    req.ID,
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
