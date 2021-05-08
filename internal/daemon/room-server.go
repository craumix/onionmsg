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

			room := data.Rooms[id]
			if room == nil {
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
					dconn.WriteInt(1)
					dconn.Flush()
					continue
				}

				log.Printf("Msg for room %s with content \"%s\"\n", id, string(msg.Content))

				room.LogMessage(msg)

				dconn.WriteInt(0)
				dconn.Flush()
			}
		}()
	}
}

func readMessage(dconn *sio.DataConn, room *types.Room) (*types.Message, error) {
	sigSalt := make([]byte, 16)
	rand.Read(sigSalt)
	_, err := dconn.WriteBytes(sigSalt)
	if err != nil {
		return nil, err
	}

	rawMeta, err := dconn.ReadBytes()
	if err != nil {
		return nil, err
	}

	meta := types.MessageMeta{}
	err = json.Unmarshal(rawMeta, &meta)
	if err != nil {
		return nil, err
	}

	sender := room.PeerByFingerprint(meta.Sender)
	if sender == nil {
		return nil, fmt.Errorf("no peer with fingerprint %s in room %s", meta.Sender, room.ID)
	}

	sig, err := dconn.ReadBytes()
	if err != nil {
		return nil, err
	}

	if !sender.Verify(append(sigSalt, rawMeta...), sig) {
		return nil, fmt.Errorf("invalid sig for meta of message from %s", meta.Sender)
	}

	var content []byte

	hash := sha256.New()
	hash.Write(sig)
	if meta.Type != types.MTYPE_BLOB {
		content, err = dconn.ReadBytes()
		if err != nil {
			return nil, err
		}

		hash.Write(content)
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
			buf, err := dconn.ReadBytes()
			if err != nil {
				return nil, err
			}

			_, err = file.Write(buf)
			if err != nil {
				return nil, err
			}

			hash.Write(buf)
		}
	}

	sig, err = dconn.ReadBytes()
	if err != nil {
		return nil, err
	}

	if !sender.Verify(hash.Sum(nil), sig) {
		return nil, fmt.Errorf("invalid sig for message from %s", meta.Sender)
	}

	return &types.Message{
		Meta:    meta,
		Content: content,
	}, nil
}
