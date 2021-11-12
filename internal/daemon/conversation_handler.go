package daemon

import (
	"encoding/json"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
)

func (d *Daemon) convClientHandler(conn net.Conn) {
	mConn := d.ConnectionManager.UseConnection(conn)
	defer mConn.Close()

	fingerprint, err := mConn.ReadFingerprintWithChallenge()
	if err != nil {
		log.WithError(err).Debug()
		mConn.SendStatusMessage(types.AuthFailed)
		return
	}

	roomID, err := mConn.ReadUUID()
	if err != nil {
		log.WithError(err).Debug()
		mConn.SendStatusMessage(types.MalformedUUID)
		return
	}

	room, ok := d.GetRoomByID(roomID.String())
	if !ok {
		log.WithField("room", roomID).Debug("unknown room")
		mConn.SendStatusMessage(types.AuthFailed)
		return
	}

	if _, ok := room.PeerByFingerprint(fingerprint); !ok {
		df := log.Fields{
			"peer": fingerprint,
			"room": roomID,
		}
		log.WithFields(df).Debug("peer is not part of room")
		mConn.SendStatusMessage(types.AuthFailed)
		return
	}

	mConn.SendStatusMessage(types.AuthOK)
	mConn.SendSyncMap(room.SyncState)

	newMsgs, _ := mConn.ReadMessages()

	for _, msg := range newMsgs {
		if !msg.SigIsValid() {
			raw, _ := json.Marshal(msg)
			log.WithField("message", string(raw)).Debug("signature is not valid")
			mConn.SendStatusMessage(types.MessageSigInvalid)
			return
		}
	}

	mConn.SendStatusMessage(types.MessagesOK)

	err = mConn.ReadAndCreateBlobs(d.BlobManager)
	if err != nil {
		log.WithError(err).Debug()
	}

	room.PushMessages(newMsgs...)

	mConn.SendStatusMessage(types.SyncOK)
}
