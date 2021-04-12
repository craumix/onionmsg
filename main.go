package main

import (
	"log"
	"time"

	"github.com/Craumix/tormsg/internal/tor"
)

func main() {
	go func() {
		err := tor.Run(true)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}()

	for (true) {
		time.Sleep(time.Second * 1)
	}
}