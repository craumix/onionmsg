package daemon

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/google/uuid"
)

func startRoomServer() error {
	server, err := net.Listen("tcp", "localhost:"+strconv.Itoa(conversationPort))
	if err != nil {
		return err
	}
	defer server.Close()

	for {
		c, err := server.Accept()
		if err != nil {
			log.Println(err)
		}

		go func() {
			dconn := sio.NewDataIO(c)
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
					continue
				}

				log.Printf("Msg for room %s with content \"%s\"\n", id, string(msg.Content))

				room.LogMessage(msg)
			}
		}()
	}
}

func readMessage(dconn *sio.DataConn, room *types.Room) (*types.Message, error) {
	sigSalt, err := writeRandom(dconn, 16)
	if err != nil {
		return nil, err
	}

	senderFP, err := dconn.ReadString()
	if err != nil {
		return nil, err
	}

	sender := room.PeerByFingerprint(senderFP)
	if sender == nil {
		return nil, fmt.Errorf("no peer with fingerprint %s in room %s", senderFP, room.ID)
	}

	rawMeta, err := readBlock(*dconn, sender, sigSalt)
	if err != nil {
		return nil, err
	}

	meta := types.MessageMeta{}
	err = json.Unmarshal(rawMeta, &meta)
	if err != nil {
		return nil, err
	}

	if sender.Fingerprint() != meta.Sender {
		return nil, fmt.Errorf("fingerprint mismatch on msg receive %s and %s", senderFP, room.ID)
	}

	var content []byte
	if meta.Type != types.MTYPE_BLOB {
		content, err = readBlock(*dconn, sender, sigSalt)
		if err != nil {
			return nil, err
		}
	} else {
		blockcount, err := dconn.ReadInt()
		if err != nil {
			return nil, err
		}

		id, err := blobmngr.MakeBlob()
		if err != nil {
			return nil, err
		}
		content = id[:]

		file, err := blobmngr.FileFromID(id)
		if err != nil {
			return nil, err
		}

		for i := 0; i < blockcount; i++ {
			buf, err := readBlock(*dconn, sender, sigSalt)
			if err != nil {
				return nil, err
			}

			_, err = file.Write(buf)
			if err != nil {
				return nil, err
			}
		}
	}

	return &types.Message{
		Meta:    meta,
		Content: content,
	}, nil
}

func readBlock(dconn sio.DataConn, sender *types.RemoteIdentity, sigSalt []byte) ([]byte, error) {
	errCount := 0
	for {
		block, _ := dconn.ReadBytes()
		sig, _ := dconn.ReadBytes()

		hash := sha256.Sum256(block)
		if sender.Verify(append(sigSalt, hash[:]...), sig) {
			dconn.WriteString("ok")
			dconn.Flush()
			return block, nil
		}

		errCount++
		if errCount < 10 {
			dconn.WriteString("resend")
			dconn.Flush()
			continue
		} else {
			dconn.WriteString("abort")
			dconn.Flush()
			return nil, fmt.Errorf("too many erros while reading block")
		}
	}
}

func writeRandom(dconn *sio.DataConn, len int) ([]byte, error) {
	r := make([]byte, len)
	rand.Read(r)
	_, err := dconn.WriteBytes(r)
	if err != nil {
		return nil, err
	}
	dconn.Flush()

	return r, nil
}
