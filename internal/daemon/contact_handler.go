package daemon

import (
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/google/uuid"

	"github.com/craumix/onionmsg/internal/types"
)

var (
	autoAcceptRequests = false
)

func contClientHandler(c net.Conn) {
	dconn := connection.WrapConnection(c)
	defer dconn.Close()

	req := &types.ContactRequest{}
	err := dconn.ReadStruct(req)
	if err != nil {
		log.Debug(err.Error())
		return
	}

	cont, ok := GetContactID(req.RemoteFP)
	if !ok {
		log.WithField("fingerprint", req.RemoteFP).Debugf("Contact Handler was adressed by unknown name")
		return
	}

	remoteID, _ := types.NewIdentity(types.Remote, req.LocalFP)
	remoteID.Meta.Admin = true

	convID, _ := types.NewIdentity(types.Self, "")

	sig, err := cont.Sign(append([]byte(convID.Fingerprint()), req.ID[:]...))
	if err != nil {
		log.Warn(err.Error())
	}
	resp := &types.ContactResponse{
		ConvFP: convID.Fingerprint(),
		Sig:    sig,
	}

	_, err = dconn.WriteStruct(resp)
	if err != nil {
		log.Warn(err.Error())
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

	if autoAcceptRequests {
		acceptRoomRequest(request.ID)
	} else {
		notifyNewRequest(request)
	}
}
