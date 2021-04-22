package daemon

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Craumix/onionmsg/internal/sio"
	"github.com/Craumix/onionmsg/internal/tor"
	"github.com/Craumix/onionmsg/internal/types"
	"github.com/google/uuid"
)

/*SerializableData struct exists purely for serialaization purposes*/
type SerializableData struct {
	ContactIdentities map[string]*types.Identity `json:"contact_identities"`
	Rooms             map[uuid.UUID]*types.Room  `json:"rooms"`
}

const (
	socksPort        = 10048
	controlPort      = 10049
	contactPort      = 10050
	conversationPort = 10051
	apiPort          = 10052

	tordir         = "tordir"
	datafile       = "onionmsg.zstd"
	unixSocketName = "onionmsg.sock"

	loopback = "127.0.0.1"
)

var (
	interactive bool
	unixSocket  bool

	data = SerializableData{
		ContactIdentities: make(map[string]*types.Identity),
		Rooms:             make(map[uuid.UUID]*types.Room),
	}

	torInstance *tor.TorInstance
	apiSocket   net.Listener
)

func StartDaemon(interactiveArg, unixSocketArg bool) {
	var err error

	interactive = interactiveArg
	unixSocket = unixSocketArg

	if unixSocket {
		apiSocket, err = sio.CreateUnixSocket(unixSocketName)
	} else {
		apiSocket, err = sio.CreateTCPSocket(apiPort)
	}
	if err != nil {
		log.Fatalf(err.Error())
	}

	go startAPIServer()

	torInstance, err = tor.NewTorInstance(tordir, socksPort, controlPort)
	if err != nil {
		log.Fatalf(err.Error())
	}

	err = loadData()
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf(err.Error())
	}

	err = loadContactIdentites()
	if err != nil {
		log.Fatalf(err.Error())
	}
	err = loadRooms()
	if err != nil {
		log.Fatalf(err.Error())
	}

	createTermSignalHandler()

	go startContactServer()
	go startRoomServer()

	if interactive {
		go startInteractive()
	}
}

func saveData() (err error) {
	err = sio.SaveDataCompressed(datafile, &data)
	return
}

func loadData() (err error) {
	err = sio.LoadCompressedData(datafile, &data)
	return
}

func createTermSignalHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Printf("Received shutdown signal, exiting gracefully...")
		exitDaemon()
	}()
}

func exitDaemon() {
	torInstance.Stop()
	saveData()
	os.Exit(0)
}
