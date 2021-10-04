package main

import (
	"flag"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/craumix/onionmsg/internal/api"
	"github.com/craumix/onionmsg/internal/daemon"
)

var (
	interactive   = false
	useUnixSocket = false
	baseDir       = "alliumd"
	portOffset    = 0
	noControlPass = false
	autoAccept    = false
	trace = false
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		TimestampFormat: "15:04:05",
	})
}

func main() {
	flag.BoolVar(&interactive, "i", interactive, "Start interactive mode")
	flag.BoolVar(&useUnixSocket, "u", useUnixSocket, "Whether to use a unix socket for the API")
	flag.StringVar(&baseDir, "d", baseDir, "The base directory for the daemons files")
	flag.IntVar(&portOffset, "o", portOffset, "The Offset for all the ports used")
	flag.BoolVar(&noControlPass, "no-pass", noControlPass, "Disable the usage of a password for the Tor Control-Port")
	flag.BoolVar(&autoAccept, "auto-accept", autoAccept, "Accept invitations automatically")
	flag.BoolVar(&trace, "trace", trace, "Set Log-Level to Trace")
	flag.Parse()

	if trace {
		log.SetLevel(log.TraceLevel)
	}

	daemon.StartDaemon(daemon.Config{
		Interactive:    interactive,
		BaseDir:        baseDir,
		PortOffset:     portOffset,
		UseControlPass: !noControlPass,
		AutoAccept:     autoAccept,
	})
	api.Start(useUnixSocket, portOffset)

	for {
		time.Sleep(time.Second * 10)
	}
}
