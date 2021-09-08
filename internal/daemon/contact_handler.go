package daemon

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/craumix/onionmsg/pkg/sio/connection"

	"github.com/craumix/onionmsg/pkg/types"
)

func contClientHandler(c net.Conn) {
	dconn := connection.WrapConnection(c)
	defer dconn.Close()

	req := &types.ContactRequest{}
	err := dconn.ReadStruct(req)
	if err != nil {
		log.Println(err.Error())
		return
	}

	cont, ok := GetContactID(req.RemoteFP)
	if !ok {
		log.Printf("Contact id %s unknown\n", req.RemoteFP)
		return
	}

	remoteID, _ := types.NewIdentity(types.Remote, req.LocalFP)

	convID, _ := types.NewIdentity(types.Self, "")

	signed, err := cont.Sign(append([]byte(convID.Fingerprint()), req.ID[:]...))
	if err != nil {
		fmt.Print(err.Error())
	}
	resp := &types.ContactResponse{
		ConvFP: convID.Fingerprint(),
		Sig:    signed,
	}

	_, err = dconn.WriteStruct(resp)
	if err != nil {
		log.Println(err.Error())
		return
	}

	dconn.Flush()

	room := &types.Room{
		Self:      convID,
		Peers:     []*types.MessagingPeer{types.NewMessagingPeer(remoteID)},
		ID:        req.ID,
		SyncState: make(types.SyncMap),
	}
	room.SetContext(context.Background())

	err = registerRoom(room)
	if err != nil {
		log.Println()
	}

	room.RunMessageQueueForAllPeers()

	//Kinda breaks interactive
	//log.Printf("Exchange succesfull uuid %s sent id %s", id, convID.Fingerprint())

}
