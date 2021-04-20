package daemon

import (
	"encoding/json"
	"log"
	"net"
	"strconv"

	"github.com/Craumix/tormsg/internal/sio"
	"github.com/Craumix/tormsg/internal/types"
	"github.com/google/uuid"
)

func startRoomServer() error {
	server, err := net.Listen("tcp", "localhost:" + strconv.Itoa(conversationPort))
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

			uid, err := uuid.FromBytes(idRaw)
			if err != nil {
				log.Println(err.Error())
				return
			}

			room := data.Rooms[uid]
			if room == nil {
				log.Printf("Unknown room with %s\n", uid)
				dconn.WriteBytes([]byte{0x01})
				dconn.Flush()
				return
			}

			amount, err := dconn.ReadInt()
			if err != nil {
				log.Println(err.Error())
				return
			}

			for i := 0; i < amount; i++ {
				raw, err := dconn.ReadBytes()
				if err != nil {
					log.Println(err.Error())
					return
				}

				msg := &types.Message{}
				err = json.Unmarshal(raw, msg)
				if err != nil {
					log.Println(err.Error())
					return
				}

				log.Printf("Read %d for message with type %d\n", len(raw), msg.Type)
				log.Printf("For room %s with content \"%s\"\n", uid, string(msg.Content))

				room.Messages = append(room.Messages, msg)
			}

			dconn.WriteBytes([]byte{0x00})
			dconn.Flush()
		}()
	}
}