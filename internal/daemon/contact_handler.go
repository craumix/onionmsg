package daemon

import (
	"log"
	"net"

	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/google/uuid"

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

	sig, err := cont.Sign(append([]byte(convID.Fingerprint()), req.ID[:]...))
	if err != nil {
		log.Println(err.Error())
	}
	resp := &types.ContactResponse{
		ConvFP: convID.Fingerprint(),
		Sig:    sig,
	}

	_, err = dconn.WriteStruct(resp)
	if err != nil {
		log.Println(err.Error())
		return
	}

	dconn.Flush()
	
	request := &types.RoomRequest{
		Room: types.Room{
			Self:      convID,
			Peers:     []*types.MessagingPeer{types.NewMessagingPeer(remoteID)},
			ID:        req.ID,
			SyncState: make(types.SyncMap),
		},
		ViaFingerprint: cont.Fingerprint(),
		ID:             uuid.New(),
	}

	data.Requests = append(data.Requests, request)

	notifyNewRequest(request)
}
