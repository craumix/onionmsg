package daemon

import (
	"log"
	"net"
	"strconv"

	"github.com/craumix/onionmsg/pkg/types"
	"github.com/craumix/onionmsg/pkg/sio"
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

			uid, err := uuid.FromBytes(idRaw)
			if err != nil {
				log.Println(err.Error())
				return
			}

			room := data.Rooms[uid]
			if room == nil {
				log.Printf("Unknown room with %s\n", uid)
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
					dconn.WriteBytes([]byte{0x01})
					dconn.Flush()
					continue
				}

				msg, err := types.MessageFromRealContentJSON(raw)
				if err != nil {
					log.Println(err.Error())
					dconn.WriteBytes([]byte{0x01})
					dconn.Flush()
					continue
				}

				sender := room.PeerByFingerprint(msg.Sender)
				if sender == nil {
					log.Printf("Received invalid sender fingerprint %s\n", msg.Sender)
					dconn.WriteBytes([]byte{0x01})
					dconn.Flush()
					continue
				}

				if !msg.Verify(sender.Pub) {
					log.Printf("Received invalid message signature for room %s\n", uid)
					dconn.WriteBytes([]byte{0x01})
					dconn.Flush()
					continue
				}

				log.Printf("Read %d for message with type %d\n", len(raw), msg.Type)
				log.Printf("For room %s with content \"%s\"\n", uid, string(msg.GetContent()))

				room.LogMessage(msg)

				dconn.WriteBytes([]byte{0x00})
				dconn.Flush()
			}
		}()
	}
}
