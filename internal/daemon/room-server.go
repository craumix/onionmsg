package daemon

import (
	"encoding/json"
	"log"
	"net"
	"strconv"
	"strings"

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

				sender := room.PeerByFingerprint(msg.Sender)
				if sender == nil {
					log.Printf("Received invalid sender fingerprint %s\n", msg.Sender)
					dconn.WriteBytes([]byte{0x01})
					dconn.Flush()
					return
				}

				if !msg.Verify(sender.Pub) {
					log.Printf("Received invalid message signature for room %s\n", uid)
					dconn.WriteBytes([]byte{0x01})
					dconn.Flush()
					return
				}

				log.Printf("Read %d for message with type %d\n", len(raw), msg.Type)
				log.Printf("For room %s with content \"%s\"\n", uid, string(msg.Content))

				room.Messages = append(room.Messages, msg)

				if msg.Type == types.MTYPE_CMD {
					if msg.Content != nil {
						handleCommand(string(msg.Content), sender, room)
					}
				}
			}

			dconn.WriteBytes([]byte{0x00})
			dconn.Flush()
		}()
	}
}

func handleCommand(cmd string, sender *types.RemoteIdentity, room *types.Room) {
	args := strings.Split(cmd, " ")
	switch args[0] {
	case "join":
		if len(args) < 2 {
			log.Printf("Not enough args for command \"%s\"\n", cmd)
			break
		}

		if room.PeerByFingerprint(args[1]) != nil {
			//User already added
			break;
		}

		newPeer, err := types.NewRemoteIdentity(args[1])
		if err != nil {
			log.Println(err.Error())
			break
		}

		room.Peers = append(room.Peers, newPeer)
		log.Printf("New peer %s added to room %s\n", newPeer.Fingerprint(), room.ID)
	default:
		log.Printf("Received invalid command \"%s\"\n", cmd)
	}
}