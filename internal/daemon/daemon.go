package daemon

import (
	"log"
	"net"

	"github.com/Craumix/tormsg/internal/file"
	"github.com/Craumix/tormsg/internal/server"
	"github.com/Craumix/tormsg/internal/tor"
	"github.com/Craumix/tormsg/internal/types"
	"github.com/google/uuid"
)

type SerializableData struct {
	ContactIdentities	map[string]*types.Identity	`json:"contact_identities"`
	Rooms				map[uuid.UUID]*types.Room	`json:"rooms"`
	MessageQueue		[]*types.WrappedMessage		`json:"message_queue"`		
}

const (
	socksPort 			= 9050
	controlPort 		= 9051
	contactPort 		= 10050
	conversationPort 	= 10051
	apiPort 			= 10052

	tordir 				= "tordir"
	datafile 			= "tormsg.zstd.aes"
	unixSocketName 		= "tormsg.sock"

	loopback			= "127.0.0.1"
)

var (
	internalTor	bool
	interactive	bool
	unixSocket	bool

	data = SerializableData{
		ContactIdentities: 	make(map[string]*types.Identity),
		Rooms: 				make(map[uuid.UUID]*types.Room),
		MessageQueue: 		make([]*types.WrappedMessage, 0),
	}

	torInstance	*tor.TorInstance

	apiSocket	net.Listener
)

func StartDaemon(interactive, internalTor, unixSocket bool) {
	var err error

	if(unixSocket) {
		apiSocket, err = createUnixSocket(unixSocketName)
	}else {
		apiSocket, err = createTCPSocket(apiPort)
	}
	if err != nil {
		log.Fatalf(err.Error())
	}

	torInstance, err = tor.NewTorInstance(internalTor, tordir, socksPort, controlPort)
	if err != nil {
		log.Fatalf(err.Error())
	}

	go server.StartContactServer(contactPort, data.ContactIdentities, data.Rooms)

	if interactive {
		go startInteractive()
	}
}

func saveData() (err error) {
	err = file.SaveDataCompressed(datafile, &data)
	return
}

func loadData() (err error) {
	err = file.LoadCompressedData(datafile, &data)
	return
}