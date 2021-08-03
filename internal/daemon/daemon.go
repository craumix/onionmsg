package daemon

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/craumix/onionmsg/internal/tor"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/craumix/onionmsg/pkg/types"
)

// SerializableData struct exists purely for serialization purposes
type SerializableData struct {
	ContactIdentities []types.Identity `json:"contactIdentities"`
	Rooms             []*types.Room    `json:"rooms"`
}

const (
	socksPort   = 10048
	controlPort = 10049
	loContPort  = 10050
	loConvPort  = 10051

	tordir   = "tordir"
	blobdir  = "onionblobs"
	datafile = "onionmsg.zstd"
)

var (
	interactive bool

	loadFuse bool

	data = SerializableData{}

	torInstance *tor.Instance

	// LastCommit is the first 7 letters of the last commit, injected at build time
	LastCommit = "unknown"
	// BuildVer is the Go Version used to build this program, obviously injected at build time
	BuildVer = "unknown"
)

// StartDaemon is used to start the application for creating identities and rooms.
// Also sending/receiving messages etc.
// Basically everything except the frontend API.
func StartDaemon(interactiveArg bool) {
	var err error

	interactive = interactiveArg

	defer func() {
		if err := recover(); err != nil {
			log.Printf("Something went seriously wrong:\n%sTrying to perfrom clean exit", err)
			exitDaemon()
		}
	}()
	startSignalHandler()

	if LastCommit != "unknown" || BuildVer != "unknown" {
		log.Printf("Built from #%s with %s\n", LastCommit, BuildVer)
	}

	err = blobmngr.InitializeDir(blobdir)
	if err != nil {
		log.Panicln(err.Error())
	}

	torInstance, err = tor.NewInstance(context.Background(), tor.DefaultConf)
	if err != nil {
		log.Panicln(err.Error())
	}
	sio.DataConnProxy = torInstance.Proxy

	err = loadData()
	if err != nil && !os.IsNotExist(err) {
		log.Panicln(err.Error())
	}
	for _, room := range data.Rooms {
		// TODO derive this from an actual context
		room.SetContext(context.TODO())
	}
	loadFuse = true

	err = initContIDServices()
	if err != nil {
		log.Panicln(err.Error())
	}
	err = initRooms()
	if err != nil {
		log.Panicln(err.Error())
	}

	go sio.StartLocalServer(loContPort, contClientHandler)
	go sio.StartLocalServer(loConvPort, convClientHandler)

	if interactive {
		time.Sleep(time.Millisecond * 500)
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

func startSignalHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Printf("Received shutdown signal, exiting gracefully...")
		exitDaemon()
	}()
}

func exitDaemon() {
	if torInstance != nil {
		torInstance.Stop()
	}

	if loadFuse {
		err := saveData()
		if err != nil {
			log.Println(err.Error())
		}
	}

	os.Exit(0)
}
