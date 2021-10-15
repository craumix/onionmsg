package daemon

import (
	"context"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/craumix/onionmsg/internal/types"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/sio"
	"github.com/craumix/onionmsg/pkg/tor"
)

const (
	defaultSocksPort   = 10048
	defaultControlPort = 10049

	defaultLocalControlPort      = 10050
	defaultLocalConversationPort = 10051

	torrc    = "torrc"
	torDir   = "cache/Tor"
	blobDir  = "blobs"
	datafile = "alliumd.zstd"

	// LastCommit is the first 7 letters of the last commit, injected at build time
	LastCommit = "unknown"
	// BuildVer is the Go Version used to build this program, obviously injected at build time
	BuildVer = "unknown"
)

// SerializableData struct exists purely for serialization purposes
type SerializableData struct {
	ContactIdentities []types.Identity     `json:"contactIdentities"`
	Rooms             []*types.Room        `json:"rooms"`
	Requests          []*types.RoomRequest `json:"requests"`
}

type Config struct {
	BaseDir, TorBinary         string
	PortOffset                 int
	UseControlPass, AutoAccept bool
}

type Daemon struct {
	Config Config

	data *SerializableData
	Tor  *tor.Instance

	//BlobManager blobmngr.BlobManager
	Notifier          types.Notifier
	ConnectionManager types.ConnectionManager

	ctx context.Context

	loContPort, loConvPort int
	datafile, blobDir      string
	loadFuse               bool
}

func NewDaemon(conf Config) (*Daemon, error) {
	newTorInstance, err := tor.NewInstance(tor.Config{
		SocksPort:   defaultSocksPort,
		ControlPort: defaultControlPort,
		DataDir:     torDir,
		TorRC:       torrc,
		ControlPass: conf.UseControlPass,
		Binary:      conf.TorBinary,
		StdOut: TorStringWriter{
			OnWrite: func(s string) {
				log.Trace("Tor-Out: " + s)
			},
		},
		StdErr: TorStringWriter{
			OnWrite: func(s string) {
				log.Debug("Tor-Err: " + s)
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return &Daemon{
		Config: conf,
		data:   &SerializableData{},
		Tor:    newTorInstance,
		//BlobManager: blobmngr.NewBlobManager(filepath.Join(conf.BaseDir, blobDir)),
		Notifier:   types.Notifier{},
		loadFuse:   false,
		loContPort: defaultLocalControlPort + conf.PortOffset,
		loConvPort: defaultLocalConversationPort + conf.PortOffset,
		datafile:   filepath.Join(conf.BaseDir, datafile),
		blobDir:    filepath.Join(conf.BaseDir, blobDir),
	}, nil
}

// StartDaemon is used to start the application for creating identities and rooms.
// Also sending/receiving messages etc.
// Basically everything except the frontend API.
func (d *Daemon) StartDaemon(ctx context.Context) error {
	d.ctx = ctx

	printBuildInfo()
	log.Info("Daemon is starting...")

	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Something went seriously wrong:\n%s\nTrying to perfrom clean exit!", err)
			d.exitDaemon()
		}
	}()

	d.startSignalHandler()

	if d.createBaseDirIfNotExists() {
		log.WithField("dir", d.Config.BaseDir).Debug("base directory not found, created it")
	}

	err := d.initBlobManager()
	if err != nil {
		log.WithError(err).Debug()
	}

	err = d.startTor()
	if err != nil {
		return err
	}

	err = d.loadData()
	if err != nil {
		return err
	}

	d.initHiddenServices()

	d.startConnectionHandlers()

	return nil
}

func printBuildInfo() {
	if LastCommit != "unknown" || BuildVer != "unknown" {
		log.Debugf("Built from #%s with %s\n", LastCommit, BuildVer)
	}
}

func (d *Daemon) createBaseDirIfNotExists() bool {
	if _, err := os.Stat(d.Config.BaseDir); os.IsNotExist(err) {
		os.MkdirAll(d.Config.BaseDir, 0700)
		return true
	}

	return false
}

func (d *Daemon) initBlobManager() error {
	err := blobmngr.InitializeDir(d.blobDir)
	if err != nil {
		return err
	}
	return nil
}

func (d *Daemon) startTor() error {
	err := d.Tor.Start(d.ctx)
	if err != nil {
		return err
	}

	d.ConnectionManager = types.NewConnectionManager(d.Tor.Proxy)

	lf := log.Fields{
		"pid":     d.Tor.Pid(),
		"version": d.Tor.Version(),
	}
	log.WithFields(lf).Info("Tor is running...")

	return nil
}

func (d *Daemon) loadData() error {
	err := sio.LoadCompressedData(d.datafile, d.data)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for _, room := range d.GetRooms() {
		room.SetContext(d.ctx)
		room.SetConnectionManager(d.ConnectionManager)
	}

	d.loadFuse = true
	return nil
}

func (d *Daemon) initHiddenServices() {
	err := d.initContactIDServices()
	if err != nil {
		panic(err)
	}

	err = d.initRooms()
	if err != nil {
		panic(err)
	}

	log.Infof("Loaded %d Contact IDs, and %d Rooms", len(d.data.ContactIdentities), len(d.data.Rooms))
}

func (d *Daemon) startConnectionHandlers() {
	go sio.StartLocalServer(d.loContPort, d.handleContact, func(err error) {
		log.WithError(err).Debug("error starting contact handler")
	})
	go sio.StartLocalServer(d.loConvPort, d.convClientHandler, func(err error) {
		log.WithError(err).Debug("error starting conversation handler")
	})
}

func (d *Daemon) startSignalHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("received shutdown signal, exiting gracefully...")
		d.exitDaemon()
	}()
}

func (d *Daemon) exitDaemon() {
	if d.Tor != nil {
		d.Tor.Stop()
	}

	if d.loadFuse {
		err := d.saveData()
		if err != nil {
			log.WithError(err).Error()
			//TODO save struct in case of unable to save
		}
	}

	os.Exit(0)
}

func (d *Daemon) saveData() error {
	return sio.SaveDataCompressed(d.datafile, d.data)
}

func (d *Daemon) TorInfo() interface{} {
	return d.Tor.Info()
}
