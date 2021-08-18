package daemon

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/craumix/onionmsg/pkg/sio/connection"
	"log"
	"net"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
)

func convClientHandler(c net.Conn) {
	dconn := connection.WrapConnection(c)
	defer dconn.Close()

	idRaw, err := dconn.ReadBytes()
	if err != nil {
		log.Println(err.Error())
		return
	}

	id, err := uuid.FromBytes(idRaw)
	if err != nil {
		log.Println(err.Error())
		return
	}

	room, ok := GetRoom(id)
	if !ok {
		log.Printf("Unknown room with %s\n", id)
		return
	}

	amount, err := dconn.ReadInt()
	if err != nil {
		log.Println(err.Error())
		return
	}

	for i := 0; i < amount; i++ {
		msg, err := readMessage(dconn, room)
		if err != nil {
			log.Print(err.Error())
			dconn.Close()
			break
		}

		log.Printf("Msg for room %s with content \"%s\"\n", id, string(msg.Content.Data))

		notifyNewMessage(id, msg)

		room.LogMessage(msg)
	}
}

func readMessage(dconn connection.ConnWrapper, room *types.Room) (types.Message, error) {
	sigSalt, err := writeRandom(dconn, 16)
	if err != nil {
		return types.Message{}, err
	}

	msgMarshal, _ := dconn.ReadBytes()
	sig, err := dconn.ReadBytes()
	if err != nil {
		return types.Message{}, err
	}

	msg := types.Message{}
	err = json.Unmarshal(msgMarshal, &msg)
	if err != nil {
		return types.Message{}, err
	}

	sender, ok := room.PeerByFingerprint(msg.Meta.Sender)
	if !ok || !sender.Verify(append(sigSalt, msgMarshal...), sig) {
		dconn.WriteString("invalid_mssage")
		dconn.Flush()
		return types.Message{}, fmt.Errorf("received invalid message")
	}

	dconn.WriteString("ok")
	dconn.Flush()

	if msg.ContainsBlob() {
		blockcount, err := dconn.ReadInt()
		if err != nil {
			return types.Message{}, err
		}

		id := msg.Content.Meta.BlobUUID

		file, err := blobmngr.FileFromID(id)
		if err != nil {
			return types.Message{}, err
		}

		rcvOK := false
		defer func() {
			file.Close()
			if !rcvOK {
				blobmngr.RemoveBlob(id)
			}
		}()

		for i := 0; i < blockcount; i++ {
			buf, err := readDataWithSig(dconn, sender, sigSalt)
			if err != nil {
				return types.Message{}, err
			}

			_, err = file.Write(buf)
			if err != nil {
				return types.Message{}, err
			}
		}
		rcvOK = true
	}

	return msg, nil
}

func readDataWithSig(dconn connection.ConnWrapper, sender types.RemoteIdentity, sigSalt []byte) ([]byte, error) {
	content, _ := dconn.ReadBytes()
	sig, err := dconn.ReadBytes()
	if err != nil {
		return nil, err
	}

	defer dconn.Flush()
	if !sender.Verify(append(sigSalt, content...), sig) {
		dconn.WriteString("invalid_sig")
		return nil, fmt.Errorf("invalid signature from remote")
	}

	dconn.WriteString("ok")
	return content, nil
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
