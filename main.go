package main

import (
	"log"
	"time"

	"github.com/Craumix/tormsg/internal/tor"
)

func main() {
	go func() {
		useMyTor := false
		exe := "tor"
		var err error

		if useMyTor {
			exe, err = tor.WriteTorToMemory()
			if err != nil {
				log.Fatalf(err.Error())
			}
		}

		err = tor.StartTor(exe)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}()

	for (true) {
		time.Sleep(time.Second * 1)
	}
}