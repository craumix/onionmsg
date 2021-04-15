package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Craumix/tormsg/internal/server"
	"github.com/Craumix/tormsg/internal/tor"
	"github.com/Craumix/tormsg/internal/types"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/wybiral/torgo"
	"golang.org/x/net/proxy"
)

/*SerializedData struct exists purely for serialaization purposes*/
type SerializedData struct {
	ContactIdentities	map[string]*types.Identity	`json:"contact_identities"`
	Rooms				map[uuid.UUID]*types.Room	`json:"rooms"`
	MessageQueue		[]*types.WrappedMessage		`json:"message_queue"`
}

const (
	socks = "9050"
	cont = "9051"
	dir = "tordir"
	internal = true

	contactPort = 10050

	datafile = "tormsg.zstd.aes"
)

var (
	data = SerializedData{
		ContactIdentities: 	make(map[string]*types.Identity),
		Rooms: 				make(map[uuid.UUID]*types.Room),
		MessageQueue: 		make([]*types.WrappedMessage, 0),
	}
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

	controller, err = tor.WaitForController(pw, "localhost:" + cont, time.Second, 30)
	if err != nil {
		log.Fatalln(err.Error())
	}

	v, _ := controller.GetVersion()
	log.Printf("Connected controller to tor version %s\n", v)

	go server.StartContactServer(contactPort, data.ContactIdentities)

	i := types.NewIdentity()
	registerContactIdentity(i)

	remote, _ := types.NewRemoteIdentity(i.Fingerprint())
	dialer, _ := proxy.SOCKS5("tcp", "localhost:9050", nil, nil)
	_ = remote
	_ = dialer
	room, _ := types.NewRoom([]*types.RemoteIdentity{remote}, dialer);
	data.Rooms[room.ID] = room

	//deregisterContactIdentity(i.Fingerprint())

	s, err := json.Marshal(data)
	fmt.Println(string(s))

	saveData()

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

	data.ContactIdentities[i.Fingerprint()] = i

	log.Printf("Registered contact identity %s\n", i.Fingerprint())

	return nil
}

func deregisterContactIdentity(fingerprint string) error {
	if data.ContactIdentities[fingerprint] == nil {
		return nil
	}

	i := data.ContactIdentities[fingerprint]
	err := controller.DeleteOnion(i.Service.Onion().ServiceID)
	if err != nil {
		return err
	}

	delete(data.ContactIdentities, fingerprint)

	log.Printf("Deregistered contact identity %s\n", i.Fingerprint())

	return nil
}

func saveData() error {
	file, err := os.OpenFile(datafile, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	enc, err := zstd.NewWriter(file)
	if err != nil {
		return err
	}

	s, _ := json.Marshal(data)

	_, err = enc.Write(s)
	if err != nil {
		return err
	}
	
	stat, _ := file.Stat()

	log.Printf("Written %d compressed bytes, down from %d (%.2f%%)\n", stat.Size(), len(s), (float64(stat.Size()) / float64(len(s))) * 100)

	return nil
}

func randomizePW(size int) {
	r := make([]byte, size)
	rand.Read(r)
	pw = base64.RawStdEncoding.EncodeToString(r)
}