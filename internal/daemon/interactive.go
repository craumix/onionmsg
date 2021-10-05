package daemon

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	//We use the new logger but dont really implement the changes, because this will remove
	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
	"github.com/google/uuid"
)

func startInteractive() {
	var err error
	cin := bufio.NewReader(os.Stdin)
	log.Println("Started interactive mode")

	for {
		fmt.Print("> ")
		cmd, _ := cin.ReadString('\n')
		cmd = strings.Trim(cmd, " \n")

		switch cmd {
		case "save":
			err = saveData()
			if err != nil {
				log.Println(err.Error())
				continue
			}
		case "exit":
			exitDaemon()
		case "add_cont":
			id, _ := types.NewIdentity(types.Contact, "")
			err = registerContID(id)
			if err != nil {
				log.Println(err.Error())
				continue
			}
		case "rm_cont":
			log.Println("Enter Fingerprint to remove:")
			fp, _ := cin.ReadString('\n')
			fp = strings.Trim(fp, " \n")

			err = deregisterContID(fp)
			if err != nil {
				log.Println(err.Error())
				continue
			}
		case "list_cont":
			log.Println("Contact Identities:")
			for _, e := range data.ContactIdentities {
				log.Println(e.Fingerprint())
				continue
			}
		case "list_rooms":
			for iRoom, room := range data.Rooms {
				log.Printf("Room %d: %s\n", iRoom, room.ID.String())
				for iPeer, peer := range room.Peers {
					log.Printf("\tPeer %d:\t%s\n", iPeer, peer.RIdentity.Fingerprint())
				}
				log.Printf("\tSelf:\t%s\n", room.Self.Fingerprint())
			}
		case "add_room":
			log.Println("Print Contact IDs (one per line, empty line to finish):")
			var ids []types.Identity
			for {
				peer, _ := cin.ReadString('\n')
				peer = strings.Trim(peer, " \n")

				if peer == "" {
					break
				}

				p, err := types.NewIdentity(types.Remote, peer)
				if err != nil {
					log.Println(err.Error())
					continue
				}
				ids = append(ids, p)
			}

			if len(ids) == 0 {
				log.Println("No peers given")
				continue
			}

			log.Printf("Trying to create a room with %d peers\n", len(ids))
			room, err := types.NewRoom(context.Background(), ids...)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			err = registerRoom(room)
			if err != nil {
				log.Println()
				continue
			}

			log.Printf("Room created with %s and %s\n", room.ID, room.Self.Fingerprint())
		case "send_message":
			log.Printf("Write Room UID in first line and message in second:")
			uid, _ := cin.ReadString('\n')
			uid = strings.Trim(uid, " \n")

			id, err := uuid.Parse(uid)
			if err != nil {
				log.Println("Unable to parse uid")
				continue
			}

			room, ok := GetRoom(id)
			if !ok {
				log.Println("No such room")
				continue
			}

			message, _ := cin.ReadString('\n')
			message = strings.Trim(message, " \n")

			room.SendMessageToAllPeers(types.MessageContent{
				Type: types.ContentTypeText,
				Data: []byte(message),
			})
			log.Println("Sent message!")
		case "list_messages":
			log.Println("Enter a room uid:")
			uid, _ := cin.ReadString('\n')
			uid = strings.Trim(uid, " \n")

			id, err := uuid.Parse(uid)
			if err != nil {
				log.Println("Unable to parse uid")
				continue
			}

			room, ok := GetRoom(id)
			if !ok {
				log.Println("No such room")
				continue
			}

			for _, msg := range room.Messages {
				log.Printf("From %s, at %s\n", msg.Meta.Sender, msg.Meta.Time)
				log.Printf("Type %s, Content \"%s\"\n", msg.Content.Type, string(msg.Content.Data))
			}
		case "stop_room":
			log.Println("Enter Room ID:")
			roomToStop, _ := cin.ReadString('\n')
			roomToStop = strings.Trim(roomToStop, " \n")

			if roomToStop == "" {
				break
			}

			for _, room := range data.Rooms {
				if room.ID.String() == roomToStop {
					room.StopQueues()
				}
			}
		case "stop_all_rooms":
			for _, room := range data.Rooms {
				room.StopQueues()
			}

		default:
			log.Printf("Unknown command \"%s\"\n", cmd)
		}
	}
}
