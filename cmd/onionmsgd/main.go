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
)

func main() {
	flag.BoolVar(&interactive, "i", interactive, "Start interactive mode")
	flag.BoolVar(&useUnixSocket, "u", useUnixSocket, "Wether to use a unix socket for the API")
	flag.StringVar(&baseDir, "d", baseDir, "The base directory for the daemons files")
	flag.IntVar(&portOffset, "o", portOffset, "The Offset for all the ports used (shifted by SumOfPorts (5) * Offset)")
	flag.Parse()

	daemon.StartDaemon(interactive, baseDir, portOffset)
	api.Start(useUnixSocket)

	for {
		time.Sleep(time.Second * 10)
	}
}
