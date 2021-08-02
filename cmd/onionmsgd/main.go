package main

import (
	"flag"
	"time"

	"github.com/craumix/onionmsg/internal/api"
	"github.com/craumix/onionmsg/internal/daemon"
)

var (
	interactive   bool
	useUnixSocket bool
)

func main() {
	flag.BoolVar(&interactive, "i", false, "start interactive mode")
	flag.BoolVar(&useUnixSocket, "u", false, "use a unix socket")
	flag.Parse()

	daemon.StartDaemon(interactive)
	api.Start(useUnixSocket)

	for {
		time.Sleep(time.Second * 10)
	}
}
