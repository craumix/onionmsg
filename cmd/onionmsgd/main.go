package main

import (
	"flag"
	"time"

	"github.com/craumix/onionmsg/internal/api"
	"github.com/craumix/onionmsg/internal/daemon"
)

var (
	interactive   = false
	useUnixSocket = false
	baseDir       = "alliumd"
	portOffset    = 0
	noControlPass = false
)

func main() {
	flag.BoolVar(&interactive, "i", interactive, "Start interactive mode")
	flag.BoolVar(&useUnixSocket, "u", useUnixSocket, "Whether to use a unix socket for the API")
	flag.StringVar(&baseDir, "d", baseDir, "The base directory for the daemons files")
	flag.IntVar(&portOffset, "o", portOffset, "The Offset for all the ports used")
	flag.BoolVar(&noControlPass, "no-pass", noControlPass, "Disable the usage of a password for the Tor Control-Port")
	flag.Parse()

	daemon.StartDaemon(interactive, baseDir, portOffset, !noControlPass)
	api.Start(useUnixSocket, portOffset)

	for {
		time.Sleep(time.Second * 10)
	}
}
