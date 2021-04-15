package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"time"

	"github.com/Craumix/tormsg/internal/server"
	"github.com/Craumix/tormsg/internal/tor"
	"github.com/Craumix/tormsg/internal/types"
	"github.com/wybiral/torgo"
	"golang.org/x/net/proxy"
)

const (
	lo = "127.0.0.1"

	socks = "9050"
	cont = "9051"
	dir = "tordir"
	internal = true

	contactPort = 10050
)

var (
	contactIdentities = make(map[string]*types.Identity)
	controller			*torgo.Controller

	pw string
)

func main() {
	randomizePW(64)
	
	err := tor.Run(pw, socks, cont, dir, internal)
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Printf("Tor seems to be runnning\n")

	controller, err = tor.WaitForController(pw, lo + ":" + cont, time.Second, 30)
	if err != nil {
		log.Fatalln(err.Error())
	}

	v, _ := controller.GetVersion()
	log.Printf("Connected controller to tor version %s\n", v)

	go server.StartContactServer(contactPort, contactIdentities)

	i := types.NewIdentity()
	registerContactIdentity(i)

	remote, _ := types.NewRemoteIdentity("Yb94xTxQRLUQnSfeLObOhAvSKU9nlA1DATM1LHsByek@qfhsbq5xvbvtvvlh5ju7oursopkuwyggygw6t2a3o5k3cmz57m7esoad")
	dialer, _ := proxy.SOCKS5("tcp", "localhost:9050", nil, nil)
	_ = remote
	_ = dialer
	_, _ = types.NewRoom([]*types.RemoteIdentity{remote}, dialer);

	for (true) {
		time.Sleep(time.Second * 10)
	}
}

func registerContactIdentity(i *types.Identity) error {
	service := i.Service
	service.LocalProxy(contactPort, contactPort)

	err := controller.AddOnion(service.Onion())
	if err != nil {
		return err
	}

	contactIdentities[i.Fingerprint()] = i

	log.Printf("Registered contact identity %s\n", i.Fingerprint())

	return nil
}

func deregisterContactIdentity(fingerprint string) error {
	if contactIdentities[fingerprint] == nil {
		return nil
	}

	i := contactIdentities[fingerprint]
	err := controller.DeleteOnion(i.Service.Onion().ServiceID)
	if err != nil {
		return err
	}

	contactIdentities[fingerprint] = nil

	log.Printf("Deregistered contact identity %s\n", i.Fingerprint())

	return nil
}

func randomizePW(size int) {
	r := make([]byte, size)
	rand.Read(r)
	pw = base64.RawStdEncoding.EncodeToString(r)
}