package daemon

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/types"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/craumix/onionmsg/pkg/sio/connection"
	"github.com/craumix/onionmsg/pkg/tor"
)

// SerializableData struct exists purely for serialization purposes
type SerializableData struct {
	ContactIdentities []types.Identity     `json:"contactIdentities"`
	Rooms             []*types.Room        `json:"rooms"`
	Requests          []*types.RoomRequest `json:"requests"`
}

type Config struct {
	BaseDir, TorBinary                      string
	PortOffset                              int
	UseControlPass, AutoAccept, Interactive bool
}

var (
	socksPort   = 10048
	controlPort = 10049
	loContPort  = 10050
	loConvPort  = 10051

	torrc    = "torrc"
	tordir   = "cache/tor"
	blobdir  = "blobs"
	datafile = "alliumd.zstd"

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
func StartDaemon(conf Config) {
	connection.GetConnFunc = connection.DialDataConn

	log.Info("Daemon is starting...")

	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Something went seriously wrong:\n%s\nTrying to perfrom clean exit!", err)
			exitDaemon()
		}
	}()

	startSignalHandler()

	printBuildInfo()

	parseParams(conf.BaseDir, conf.PortOffset)

	initBlobManager()

	startTor(conf.UseControlPass, conf.TorBinary)

	loadData()

	initHiddenServices()

	startConnectionHandlers(conf.AutoAccept)

	if conf.Interactive {
		time.Sleep(time.Millisecond * 500)
		go startInteractive()
	}
}

func printBuildInfo() {
	if LastCommit != "unknown" || BuildVer != "unknown" {
		log.Debugf("Built from #%s with %s\n", LastCommit, BuildVer)
	}
}

func parseParams(baseDir string, portOffset int) {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		os.MkdirAll(baseDir, 0700)
	}

	socksPort += portOffset
	controlPort += portOffset
	loContPort += portOffset
	loConvPort += portOffset

	torrc = filepath.Join(baseDir, torrc)
	tordir = filepath.Join(baseDir, tordir)
	blobdir = filepath.Join(baseDir, blobdir)
	datafile = filepath.Join(baseDir, datafile)
}

func initBlobManager() {
	err := blobmngr.InitializeDir(blobdir)
	if err != nil {
		panic(err)
	}
}

func startTor(useControlPass bool, binaryPath string) {
	var err error

	torInstance, err = tor.NewInstance(context.Background(), tor.Conf{
		SocksPort:   socksPort,
		ControlPort: controlPort,
		DataDir:     tordir,
		TorRC:       torrc,
		ControlPass: useControlPass,
		Binary:      binaryPath,
		StdOut: StringWriter{
			OnWrite: func(s string) {
				log.Trace("Tor-Out: " + s)
			},
		},
		StdErr: StringWriter{
			OnWrite: func(s string) {
				log.Debug("Tor-Err: " + s)
			},
		},
	})
	if err != nil {
		panic(err)
	}

	connection.DataConnProxy = torInstance.Proxy

	lf := log.Fields{
		"pid":     torInstance.Pid(),
		"version": torInstance.Version(),
	}
	log.WithFields(lf).Info("tor is running...")
}

func loadData() {
	err := sio.LoadCompressedData(datafile, &data)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	for _, room := range data.Rooms {
		// TODO derive this from an actual context
		room.SetContext(context.Background())
	}
	loadFuse = true
}

func initHiddenServices() {
	err := initContIDServices()
	if err != nil {
		panic(err)
	}

	err = initRooms()
	if err != nil {
		panic(err)
	}

	log.Infof("Loaded %d Contact IDs, and %d Rooms", len(data.ContactIdentities), len(data.Rooms))
}

func startConnectionHandlers(autoAccept bool) {
	autoAcceptRequests = autoAccept

	go sio.StartLocalServer(loContPort, contClientHandler, func(err error) {
		log.WithError(err).Debug("error starting contact handler")
	})
	go sio.StartLocalServer(loConvPort, convClientHandler, func(err error) {
		log.WithError(err).Debug("error starting conversation handler")
	})
}

func startSignalHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("received shutdown signal, exiting gracefully...")
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
			log.WithError(err).Error()
			//TODO save struct in case of unable to save
		}
	}

	os.Exit(0)
}

func saveData() error {
	return sio.SaveDataCompressed(datafile, &data)
}
