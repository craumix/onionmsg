package types

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Craumix/onionmsg/internal/sio"
	"github.com/google/uuid"
	"golang.org/x/net/proxy"
)

const (
	queueTimeout = time.Second * 3
)

type RemoteIdentity struct {
	Pub     ed25519.PublicKey `json:"public_key"`
	Service string            `json:"service"`
	Queue   []*Message        `json:"queue"`
}

func NewRemoteIdentity(fingerprint string) (*RemoteIdentity, error) {
	if !strings.Contains(fingerprint, "@") {
		return nil, fmt.Errorf("%s is not a valid id", fingerprint)
	}

	tmp := strings.Split(fingerprint, "@")
	k, err := base64.RawURLEncoding.DecodeString(tmp[0])
	if err != nil {
		return nil, err
	}

	return &RemoteIdentity{
		Pub:     ed25519.PublicKey(k),
		Service: tmp[1],
	}, nil
}

func (i *RemoteIdentity) Verify(msg, sig []byte) bool {
	return ed25519.Verify(i.Pub, msg, sig)
}

func (i *RemoteIdentity) URL() string {
	return i.Service + ".onion"
}

func (i *RemoteIdentity) Fingerprint() string {
	return i.B64PubKey() + "@" + i.Service
}

func (i *RemoteIdentity) B64PubKey() string {
	return base64.RawURLEncoding.EncodeToString(i.Pub)
}

func (i *RemoteIdentity) RunMessageQueue(dialer proxy.Dialer, conversationPort int, roomID uuid.UUID) {
	for {
		if len(i.Queue) == 0 {
			time.Sleep(queueTimeout)
			continue
		}

		conn, err := dialer.Dial("tcp", i.URL()+":"+strconv.Itoa(conversationPort))
		if err != nil {
			//Expected error
			//log.Println(err.Error())
		} else {
			dconn := sio.NewDataIO(conn)

			dconn.WriteBytes(roomID[:])
			dconn.WriteInt(len(i.Queue))
			dconn.Flush()
			for index, msg := range i.Queue {
				raw, _ := json.Marshal(msg)

				_, err = dconn.WriteBytes(raw)
				if err != nil {
					fmt.Println(err.Error())
					break
				}
				dconn.Flush()

				state, err := dconn.ReadBytes()
				if err != nil {
					fmt.Println(err.Error())
					break
				}

				if state[0] != 0x00 {
					log.Printf("Received invalid state for message %d\n", state[0])
				}

				copy(i.Queue[index:], i.Queue[index+1:]) // Shift a[i+1:] left one index.
				i.Queue[len(i.Queue)-1] = nil            // Erase last element (write zero value).
				i.Queue = i.Queue[:len(i.Queue)-1]       // Truncate slice.
			}
		}

		time.Sleep(queueTimeout)
	}
}
