package daemon

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"

	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/google/uuid"

	"github.com/craumix/onionmsg/internal/types"
)

func (d *Daemon) handleContact(conn net.Conn) {
	dataConn := connection.WrapConnection(conn)
	defer dataConn.Close()

	cReq, err := readContactRequest(dataConn)
	if err != nil {
		log.WithError(err).Debug()
		return
	}

	cID, found := d.GetContactIDByFingerprint(cReq.RemoteFP)
	if !found {
		log.WithError(fmt.Errorf("contact handler was addressed by unknown name")).Debug()
		return
	}

	rReq, err := writeContactResponse(dataConn, cReq, cID)
	if err != nil {
		log.WithError(err).Debug()
		return
	}

	dataConn.Flush()

	d.AddRoomRequest(rReq)

	if d.Config.AutoAccept {
		d.AcceptRoomRequest(rReq.ID.String())
	} else {
		d.Notifier.NotifyNewRequest(rReq)
	}
}

func writeContactResponse(dataConn connection.ConnWrapper, cReq *types.ContactRequest, cID types.Identity) (*types.RoomRequest, error) {
	remoteID, _ := types.NewIdentity(types.Remote, cReq.LocalFP)
	remoteID.Meta.Admin = true

	convID, _ := types.NewIdentity(types.Self, "")

	sig, err := cID.Sign(append([]byte(convID.Fingerprint()), cReq.ID[:]...))
	if err != nil {
		return nil, err
	}
	resp := &types.ContactResponse{
		ConvFP: convID.Fingerprint(),
		Sig:    sig,
	}

	_, err = dataConn.WriteStruct(resp)
	if err != nil {
		return nil, err
	}

	return &types.RoomRequest{
		Room: types.Room{
			Self:      convID,
			Peers:     []*types.MessagingPeer{types.NewMessagingPeer(remoteID)},
			ID:        cReq.ID,
			SyncState: make(types.SyncMap),
		},
		ViaFingerprint: cID.Fingerprint(),
		ID:             uuid.New(),
	}, nil
}

func readContactRequest(dataConn connection.ConnWrapper) (*types.ContactRequest, error) {
	cReq := &types.ContactRequest{}
	err := dataConn.ReadStruct(cReq)
	if err != nil {
		return nil, err
	}

	return cReq, err
}
