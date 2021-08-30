package daemon

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/craumix/onionmsg/pkg/sio/connection"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
)

func convClientHandler(c net.Conn) {
	conn := connection.WrapConnection(c)
	defer conn.Close()

	fingerprint, err := readFingerprintWithChallenge(conn)
	if err != nil {
		log.Println(err.Error())
		conn.WriteString("auth_failed")
		return
	}

	idRaw, err := conn.ReadBytes(false)
	if err != nil {
		log.Println(err.Error())
		return
	}

	id, err := uuid.FromBytes(idRaw)
	if err != nil {
		log.Println(err.Error())
		conn.WriteString("malformed_uuid")
		return
	}

	room, ok := GetRoom(id)
	if !ok {
		log.Printf("Unknown room with %s\n", id)
		conn.WriteString("auth_failed")
		return
	}

	if _, ok := room.PeerByFingerprint(fingerprint); !ok {
		log.Printf("Peer %s is not part of room %s", fingerprint, id)
		conn.WriteString("auth_failed")
		return
	}

	conn.WriteString("auth_ok")
	conn.WriteStruct(room.SyncState, false)
	conn.Flush()

	newMsgs := make([]types.Message, 0)
	conn.ReadStruct(&newMsgs, true)

	for _, msg := range newMsgs {
		if !msg.SigIsValid() {
			raw, _ := json.Marshal(msg)
			log.Printf("Sig for %s is not valid", string(raw))
			conn.WriteString("message_sig_invalid " + string(raw))
			return
		}
	}

	if len(newMsgs) > 0 {
		notifyNewMessages(id, newMsgs...)
	}

	conn.WriteString("messages_ok")
	conn.Flush()

	err = readBlobs(conn)
	if err != nil {
		log.Println(err.Error())
	}

	room.PushMessages(newMsgs...)

	conn.WriteString("sync_ok")
	conn.Flush()
}

func readBlobs(conn connection.ConnWrapper) error {
	ids := make([]uuid.UUID, 0)
	conn.ReadStruct(&ids, false)

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
			buf, err := conn.ReadBytes(false)
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
	_, err := dconn.WriteBytes(r, false)
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
	sig, err := conn.ReadBytes(false)
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
