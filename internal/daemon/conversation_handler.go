package daemon

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
)

func convClientHandler(c net.Conn) {
	dconn := sio.WrapConnection(c)
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

		log.Printf("Msg for room %s with content \"%s\"\n", id, string(msg.Content))

		room.LogMessage(msg)
	}
}

func readMessage(dconn *sio.DataConn, room *types.Room) (types.Message, error) {
	sigSalt, err := writeRandom(dconn, 16)
	if err != nil {
		return types.Message{}, err
	}

	rawMeta, _ := dconn.ReadBytes()
	sig, err := dconn.ReadBytes()
	if err != nil {
		return types.Message{}, err
	}

	meta := types.MessageMeta{}
	err = json.Unmarshal(rawMeta, &meta)
	if err != nil {
		return types.Message{}, err
	}

	sender, ok := room.PeerByFingerprint(meta.Sender)
	if !ok || !sender.Verify(append(sigSalt, rawMeta...), sig) {
		dconn.WriteString("invalid_meta")
		dconn.Flush()
		return types.Message{}, fmt.Errorf("received invalid meta for message")
	}

	dconn.WriteString("ok")
	dconn.Flush()

	var content []byte
	if meta.Type != types.MTYPE_BLOB {
		content, err = readDataWithSig(*dconn, sender, sigSalt)
		if err != nil {
			return types.Message{}, err
		}
	} else {
		blockcount, err := dconn.ReadInt()
		if err != nil {
			return types.Message{}, err
		}

		id, err := blobmngr.MakeBlob()
		if err != nil {
			return types.Message{}, err
		}
		content = id[:]

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
			buf, err := readDataWithSig(*dconn, sender, sigSalt)
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

	return types.Message{
		Meta:    meta,
		Content: content,
	}, nil
}

func readDataWithSig(dconn sio.DataConn, sender types.RemoteIdentity, sigSalt []byte) ([]byte, error) {
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

func writeRandom(dconn *sio.DataConn, length int) ([]byte, error) {
	r := make([]byte, length)
	rand.Read(r)
	_, err := dconn.WriteBytes(r)
	if err != nil {
		return nil, err
	}
	dconn.Flush()

	return r, nil
}
