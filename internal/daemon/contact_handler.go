package daemon

import (
	"context"
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

	cont, ok := GetKey(req.RemoteFP)
	if !ok {
		log.Printf("Contact id %s unknown\n", req.RemoteFP)
		return
	}

	remoteID, _ := types.NewRemoteIdentity(req.LocalFP)

	convID := types.GenerateKey()

	resp := &types.ContactResponse{
		ConvFP: types.Fingerprint(convID),
		Sig:    types.Sign(cont, append([]byte(types.Fingerprint(convID)), req.ID[:]...)),
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
