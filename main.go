package main

import (
	"flag"
	"net"
	"os"
	"time"

	"github.com/Craumix/tormsg/internal/daemon"
	"github.com/Craumix/tormsg/internal/types"
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

var (
	data = SerializedData{
		ContactIdentities: 	make(map[string]*types.Identity),
		Rooms: 				make(map[uuid.UUID]*types.Room),
		MessageQueue: 		make([]*types.WrappedMessage, 0),
	}

	torproc		*os.Process
	controller	*torgo.Controller
	dialer		proxy.Dialer

	apiSocket	net.Listener
)

func main() {
	externalTor 	:= flag.Bool("e", false, "use external tor")
	interactive 	:= flag.Bool("i", false, "start interactive mode")
	useUnixSocket 	:= flag.Bool("u", false, "use a unix socket")
	flag.Parse()

	daemon.StartDaemon(*interactive, !*externalTor, *useUnixSocket)

	for {
		time.Sleep(time.Second * 10)
	}
}