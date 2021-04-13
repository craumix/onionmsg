package main

import (
	"crypto/rand"
	"encoding/base64"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/Craumix/tormsg/internal/tor"
	"github.com/Craumix/tormsg/internal/types"
)

const (
	lo = "127.0.0.1"

	socks = "9050"
	cont = "9051"
	dir = "tordir"
	internal = true
)

var (
	pw string
)

func main() {
	randomizePW(64)
	
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

	i := types.NewIdentity()
	log.Printf("ID: %s\n", i.Fingerprint())

	/*
	img, err := i.QR(256)
	if err != nil {
		log.Println(err.Error())
	}else {
		f, err := os.OpenFile(dir + "/id.png", os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Println(err.Error())
		}
		png.Encode(f, img)
		f.Close()
	}
	*/
	
	for (true) {
		time.Sleep(time.Second * 10)
	}
}

func randomizePW(size int) {
	r := make([]byte, size)
	rand.Read(r)
	pw = base64.RawStdEncoding.EncodeToString(r)
}