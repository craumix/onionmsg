package daemon

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Craumix/tormsg/internal/types"
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

		switch(cmd) {
		/*
		case "load":
			err = loadData()
			if err != nil {
				log.Println(err.Error())
				continue
			}
			err = torInstance.Stop()
			if err != nil {
				log.Println(err.Error())
				continue
			}
			torInstance, err = tor.NewTorInstance(internalTor, tordir, socksPort, controlPort)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			err = loadContactIdentites()
			if err != nil {
				log.Println(err.Error())
				continue
			}
			runMessageQueues()
		*/
		case "save":
			err = saveData()
			if err != nil {
				log.Println(err.Error())
				continue
			}
		case "exit":
			torInstance.Stop()
			saveData()
			os.Exit(0)
		case "add_cont":
			err = registerContactIdentity(types.NewIdentity())
			if err != nil {
				log.Println(err.Error())
				continue
			}
		case "rm_cont":
			log.Println("Enter Fingerprint to remove:")
			fp, _ := cin.ReadString('\n')
			fp = strings.Trim(fp, " \n")

			err = deregisterContactIdentity(fp)
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
			log.Println("Rooms:")
			for _, e := range data.Rooms {
				log.Printf("%s with %d peers\nSelf: %s\n", e.ID, len(e.Peers), e.Self.Fingerprint())
			}
		case "add_room":
			log.Println("Print Contact IDs (one per line, empty line to finish):")
			ids := make([]*types.RemoteIdentity, 0)
			for {
				peer, _ := cin.ReadString('\n')
				peer = strings.Trim(peer, " \n")

				if peer == "" {
					break;
				}
				
				p, err := types.NewRemoteIdentity(peer)
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
			room, err := types.NewRoom(ids, torInstance.Proxy, contactPort, conversationPort)
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
					log.Println()
					continue
				}

				room := data.Rooms[id]
				if room == nil {
					log.Println("No such room")
					continue
				}

				message, _ := cin.ReadString('\n')
				message = strings.Trim(message, " \n")

				room.SendMessage(types.MTYPE_TEXT, []byte(message))
				log.Println("Sent message!")
		default:
			log.Printf("Unknown command \"%s\"\n", cmd)
		}
	}
}