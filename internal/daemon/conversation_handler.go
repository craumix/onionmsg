package daemon

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/pkg/sio/connection"

	"github.com/craumix/onionmsg/internal/types"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/google/uuid"
)

func (d *Daemon) convClientHandler(c net.Conn) {
	conn := connection.WrapConnection(c)
	defer conn.Close()

	fingerprint, err := readFingerprintWithChallenge(conn)
	if err != nil {
		log.WithError(err).Debug()
		conn.WriteString("auth_failed")
		return
	}

	idRaw, err := conn.ReadBytes()
	if err != nil {
		log.WithError(err).Debug()
		return
	}

	id, err := uuid.FromBytes(idRaw)
	if err != nil {
		log.WithError(err).Debug()
		conn.WriteString("malformed_uuid")
		return
	}

	room, ok := d.GetRoom(id)
	if !ok {
		log.WithField("room", id).Debug("unknown room")
		conn.WriteString("auth_failed")
		return
	}

	if _, ok := room.PeerByFingerprint(fingerprint); !ok {
		df := log.Fields{
			"peer": fingerprint,
			"room": id,
		}
		log.WithFields(df).Debug("peer is not part of room")
		conn.WriteString("auth_failed")
		return
	}

	conn.WriteString("auth_ok")
	conn.WriteStruct(room.SyncState)
	conn.Flush()

	newMsgs := make([]types.Message, 0)
	conn.ReadStruct(&newMsgs)

	for _, msg := range newMsgs {
		if !msg.SigIsValid() {
			raw, _ := json.Marshal(msg)
			log.WithField("message", string(raw)).Debug("signature is not valid")
			conn.WriteString("message_sig_invalid " + string(raw))
			return
		}
	}

	conn.WriteString("messages_ok")
	conn.Flush()

	err = readBlobs(conn)
	if err != nil {
		log.WithError(err).Debug()
	}

	room.PushMessages(newMsgs...)

	if len(newMsgs) > 0 {
		d.Notifier.NotifyNewMessage(id, newMsgs...)
	}

	conn.WriteString("sync_ok")
	conn.Flush()
}

func readBlobs(conn connection.ConnWrapper) error {
	ids := make([]uuid.UUID, 0)
	conn.ReadStruct(&ids)

	for _, id := range ids {
		blockcount, err := conn.ReadInt()
		if err != nil {
			return err
		}

		file, err := blobmngr.FileFromID(id)
		if err != nil {
			return err
		}

		rcvOK := false
		defer func() {
			file.Close()
			if !rcvOK {
				blobmngr.RemoveBlob(id)
			}
		}()

		for i := 0; i < blockcount; i++ {
			buf, err := conn.ReadBytes()
			if err != nil {
				return err
			}

			_, err = file.Write(buf)
			if err != nil {
				return err
			}

			conn.WriteString("block_ok")
			conn.Flush()
		}

		conn.WriteString("blob_ok")
		conn.Flush()

		rcvOK = true
	}

	return nil
}

func writeRandom(dconn connection.ConnWrapper, length int) ([]byte, error) {
	r := make([]byte, length)
	rand.Read(r)
	_, err := dconn.WriteBytes(r)
	if err != nil {
		return nil, err
	}
	dconn.Flush()

	return r, nil
}

func readFingerprintWithChallenge(conn connection.ConnWrapper) (string, error) {
	challenge, _ := writeRandom(conn, 32)

	fingerprint, err := conn.ReadString()
	if err != nil {
		return "", err
	}
	sig, err := conn.ReadBytes()
	if err != nil {
		return "", err
	}

	keyBytes, err := base64.RawURLEncoding.DecodeString(fingerprint)
	if err != nil {
		return "", err
	}

	key := ed25519.PublicKey(keyBytes)
	if !ed25519.Verify(key, challenge, sig) {
		return "", fmt.Errorf("remote failed challenge")
	}

	return fingerprint, nil
}
