package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Craumix/tormsg/internal/server"
	"github.com/Craumix/tormsg/internal/tor"
	"github.com/Craumix/tormsg/internal/types"
	"github.com/DataDog/zstd"
	"github.com/google/uuid"
	"github.com/wybiral/torgo"
	"golang.org/x/net/proxy"
)

/*SerializedData struct exists purely for serialaization purposes*/
type SerializedData struct {
	ContactIdentities	map[string]*types.Identity	`json:"contact_identities"`
	Rooms				map[uuid.UUID]*types.Room	`json:"rooms"`
	MessageQueue		[]*types.WrappedMessage		`json:"message_queue"`		
}

const (
	
	tordir = "tordir"

	socksPort 			= 9050
	controlPort 		= 9051
	contactPort 		= 10050
	conversationPort 	= 10051
	apiPort 			= 10052

	unixSocketName 		= "tormsg.sock"
	datafile 			= "tormsg.zstd.aes"
)

var (
	data = SerializedData{
		ContactIdentities: 	make(map[string]*types.Identity),
		Rooms: 				make(map[uuid.UUID]*types.Room),
		MessageQueue: 		make([]*types.WrappedMessage, 0),
	}

	torproc		*os.Process
	controller	*torgo.Controller
	dialer		proxy.Dialer

	externalTor		*bool
	interactive 	*bool
	useUnixSocket 	*bool

	apiSocket	net.Listener
)

func main() {
	externalTor 	= flag.Bool("e", false, "use external tor")
	interactive 	= flag.Bool("i", false, "start interactive mode")
	useUnixSocket 	= flag.Bool("u", false, "use a unix socket")
	flag.Parse()

	err := createApiSocket()
	if err != nil {
		log.Fatalf(err.Error())
	}

	if _, err = os.Stat(datafile); err == nil {
		loadData()
	}

	err = startTor()
	if err != nil {
		log.Fatalln(err.Error())
	}
	err = loadContactIdentites()
	if err != nil {
		log.Fatalln(err.Error())
	}

	go server.StartContactServer(contactPort, data.ContactIdentities, data.Rooms)

	/*
	i := types.NewIdentity()
	registerContactIdentity(i)

	remote, _ := types.NewRemoteIdentity(i.Fingerprint())
	_ = remote
	_ = dialer
	room, _ := types.NewRoom([]*types.RemoteIdentity{remote}, dialer);
	data.Rooms[room.ID] = room

	//deregisterContactIdentity(i.Fingerprint())

	err = saveData()
	if err != nil {
		fmt.Println(err.Error())
	}
	*/

	if *interactive {
		go startInteractive()
	}

	for {
		time.Sleep(time.Second * 10)
	}
}

func createApiSocket() (err error) {
	if(*useUnixSocket) {
		if(runtime.GOOS == "linux") {
			runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
			if(runtimeDir == "") {
				runtimeDir = "/tmp"
				log.Printf("Unable to determine Env XDG_RUNTIME_DIR, using %s\n", runtimeDir)
			}

			socketPath := runtimeDir + "/" + unixSocketName;
			log.Printf("Using unix socket with path %s\n", socketPath)
			
			if _, ferr := os.Stat(socketPath); ferr == nil {
  				log.Printf("Unix socket already exists, removing")
				err = os.Remove(socketPath)
				if err != nil {
					return
				}
			}

			apiSocket, err = net.Listen("unix", socketPath)
			return
		}
		err = fmt.Errorf("Cannot use unix socket on %s", runtime.GOOS)
		return
	}
	address := "localhost:" + strconv.Itoa(apiPort)
	log.Printf("Listening on %s\n", address)
	apiSocket, err = net.Listen("tcp", address)
	return
}

func startTor() (err error) {
	err = os.RemoveAll(tordir)
	if err != nil {
		return
	}

	pw := randomString(64)
	torproc, err = tor.Run(pw, tordir, socksPort, controlPort, !*externalTor)
	if err != nil {
		return
	}

	log.Printf("Tor seems to be runnning\n")

	controller, err = tor.WaitForController(pw, "127.0.0.1:" + strconv.Itoa(controlPort), time.Second, 30)
	if err != nil {
		return
	}

	v, _ := controller.GetVersion()
	log.Printf("Connected controller to tor version %s\n", v)

	dialer, _ = proxy.SOCKS5("tcp", "127.0.0.1:" + strconv.Itoa(socksPort), nil, nil)

	return
}

func stopTor() (err error) {
	if torproc != nil {
		if runtime.GOOS == "windows" {
			err = torproc.Kill()
		}else {
			err = torproc.Signal(os.Interrupt)
		}
	}

	//Give Tor some time to stop and drop file locks
	time.Sleep(time.Millisecond * 500)

	err = os.RemoveAll(tordir)
	if err != nil {
		return
	}

	return
}

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
			err = stopTor()
			if err != nil {
				log.Println(err.Error())
				continue
			}
			err = startTor()
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
			stopTor()
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

			room, err := types.NewRoom(ids, dialer)
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

func loadContactIdentites() (err error) {
	for _, i := range data.ContactIdentities {
		s := i.Service
		s.LocalProxy(contactPort, contactPort)

		err = controller.AddOnion(s.Onion())
		if err != nil {
			return
		}
	}

	log.Printf("Loaded %d Contact Identities\n", len(data.ContactIdentities))

	return
}

func registerContactIdentity(i *types.Identity) error {
	service := i.Service
	service.LocalProxy(contactPort, contactPort)

	err := controller.AddOnion(service.Onion())
	if err != nil {
		return err
	}

	data.ContactIdentities[i.Fingerprint()] = i

	log.Printf("Registered contact identity %s\n", i.Fingerprint())

	return nil
}

func deregisterContactIdentity(fingerprint string) error {
	if data.ContactIdentities[fingerprint] == nil {
		return nil
	}

	i := data.ContactIdentities[fingerprint]
	err := controller.DeleteOnion(i.Service.Onion().ServiceID)
	if err != nil {
		return err
	}

	delete(data.ContactIdentities, fingerprint)

	log.Printf("Deregistered contact identity %s\n", i.Fingerprint())

	return nil
}

func saveData() error {
	file, err := os.OpenFile(datafile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	raw, _ := json.Marshal(data)
	comp, _ := zstd.Compress(nil, raw)

	_, err = file.Write(comp)
	if err != nil {
		return err
	}
	
	log.Printf("Written %d compressed bytes, was %d (%.2f%%)\n", len(comp), len(raw), (float64(len(comp)) / float64(len(raw))) * 100)

	return nil
}

func loadData() error {
	file, err := os.OpenFile(datafile, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	
	comp, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	raw, _:= zstd.Decompress(nil, comp)
	
	json.Unmarshal(raw, &data)

	log.Printf("Decoded %d bytes from file contents\n", len(raw))

	return nil
}

func randomString(size int) string {
	r := make([]byte, size)
	rand.Read(r)
	return base64.RawStdEncoding.EncodeToString(r)
}