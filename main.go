package main

import (
	"log"
	"time"

	"github.com/Craumix/tormsg/internal/tor"
	"github.com/Craumix/tormsg/internal/types"
)

const (
	lo = "127.0.0.1"

	pw = "abc"
	socks = "9050"
	cont = "9051"
	dir = "tordir"
	internal = true
)

func main() {
	err := tor.Run(pw, socks, cont, dir, internal)
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Printf("Tor seems to be runnning\n")

	ctrl, err := tor.WaitForController(pw, lo + ":" + cont, time.Second, 30)
	if err != nil {
		log.Fatalln(err.Error())
	}

	_ = ctrl
	log.Printf("Connected controller to tor\n")

	service := types.NewHiddenService()
	service.Proxy(80, "example.org")
	service.LocalProxy(8080, 8080)

	err = ctrl.AddOnion(service.Onion())
	if err != nil {
		log.Println(err.Error())
	}else {
		log.Printf("Started hidden service at %s", service.URL())
	}

	for (true) {
		time.Sleep(time.Second * 10)
	}
}