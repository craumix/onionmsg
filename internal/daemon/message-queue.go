package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/Craumix/tormsg/internal/types"
)

const (
	timeout = time.Second * 10;
)

//TODO fixme, use better sorting or smth because this has horrible runtime

func runQueueLoop() {
	log.Println("Started message queue")

	for {
		//use this in case more items get added, but the will only best sent in the next run
		queueSize 	:= len(data.MessageQueue)
		sentItems 	:= make([]bool, queueSize)
		identies	:= make(map[string]*types.RemoteIdentity)

		for i := 0; i < queueSize; i++ {
			elem := &data.MessageQueue[i]
			if identies[elem.Remote.Fingerprint()] == nil {
				identies[elem.Remote.Fingerprint()] = elem.Remote
			}
		}
		
		var wg sync.WaitGroup
		wg.Add(len(identies))
		for key, e := range identies {
			go func() {
				defer wg.Done()

				conn, err := torInstance.Proxy.Dial("tcp", e.URL() + ":" + strconv.Itoa(conversationPort))
				if err != nil {
					log.Println(err.Error())
					return
				}
				dconn := types.NewDataIO(conn)
				defer dconn.Close()
				
				msgIndexes := make([]int, 0)
				for i := 0; i < queueSize; i++ {
					if data.MessageQueue[i].Remote.Fingerprint() == key {
						msgIndexes = append(msgIndexes, i)
					}
				}
				
				dconn.WriteInt(len(msgIndexes))
				dconn.Flush()
				for _, index := range msgIndexes {
					wmsg := data.MessageQueue[index]

					raw, _ := json.Marshal(wmsg.Message)

					_, err = dconn.WriteBytes(raw)
					if err != nil {
						fmt.Println(err.Error())
						return
					}
					dconn.Flush()

					sig, err := dconn.ReadBytes()
					if err != nil {
						log.Println(err.Error())
						return
					}

					if !wmsg.Remote.Verify(raw, sig) {
						log.Printf("Signature invalid for %s\n", wmsg.Remote.Fingerprint())
						return
					}

					sentItems[index] = true
				}
			}()
		}
		wg.Wait()

		//Probably add a lock or smth
		newQueue := make([]types.WrappedMessage, 0)
		for i := 0; i < len(data.MessageQueue); i++ {
			if i > len(sentItems) || !sentItems[i] {
				newQueue = append(newQueue, data.MessageQueue[i])
			}
		}

		data.MessageQueue = newQueue

		time.Sleep(timeout)
	}
}