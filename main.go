package main

import (
	"flag"
	"time"

	"github.com/Craumix/tormsg/internal/daemon"
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