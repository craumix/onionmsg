package daemon

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Craumix/tormsg/internal/tor"
	"github.com/Craumix/tormsg/internal/types"
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

			room, err := types.NewRoom(ids, torInstance.Proxy)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			data.Rooms[room.ID] = room
			log.Printf("Room created with %s and %s\n", room.ID, room.Self.Fingerprint())
		default:
			log.Printf("Unknown command \"%s\"\n", cmd)
		}
	}
}