package types

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/net/proxy"

	"github.com/craumix/onionmsg/pkg/sio"
)

type Room struct {
	Self     *Identity         `json:"self"`
	Peers    []*RemoteIdentity `json:"peers"`
	ID       uuid.UUID         `json:"uuid"`
	Messages []*Message        `json:"messages"`
	Name     string            `json:"name"`
	Nicks    map[string]string `json:"nicks"`

	queueTerminate chan bool
}

func NewRoom(contactIdentities []*RemoteIdentity, dialer proxy.Dialer, contactPort, conversationPort int) (*Room, error) {
	s := NewIdentity()
	peers := make([]*RemoteIdentity, 0)
	id := uuid.New()

	room := &Room{
		Self:     s,
		Peers:    peers,
		ID:       id,
		Messages: make([]*Message, 0),
		Nicks:    make(map[string]string),
	}

	var sharedErr *error
	var wg sync.WaitGroup

	wg.Add(len(contactIdentities))

	for _, contact := range contactIdentities {
		go func(c *RemoteIdentity) {
			err := room.addUserWithContactID(c, dialer, contactPort)
			if err != nil {
				*sharedErr = err
			}
			wg.Done()
		}(contact)
	}

	wg.Wait()

	if *sharedErr != nil {
		return nil, *sharedErr
	}

	room.syncPeerLists()

	return room, nil
}

/*
AddUser adds a user to the Room, and if successfull syncs the PeerLists.
If not successfull returns the error.
*/
func (r *Room) AddUser(contact *RemoteIdentity, dialer proxy.Dialer, contactPort int) error {
	err := r.addUserWithContactID(contact, dialer, contactPort)
	if err != nil {
		return err
	}

	r.syncPeerLists()
	return nil
}

/*
Syncs the user list for all peers.
This only adds users, but can remove users from peers.
*/
func (r *Room) syncPeerLists() {
	for _, peer := range r.Peers {
		r.SendMessage(MTYPE_CMD, []byte("join "+peer.Fingerprint()))
	}
}

/*
This function tries to add a user with the contactID to the room.
This only adds the user, so the user lists are then out of sync.
Call syncPeerLists() to sync them again.
*/
func (r *Room) addUserWithContactID(contact *RemoteIdentity, dialer proxy.Dialer, contactPort int) error {
	conn, err := dialer.Dial("tcp", contact.URL()+":"+strconv.Itoa(contactPort))
	if err != nil {
		return err
	}

	dconn := sio.NewDataIO(conn)

	_, err = dconn.WriteString(contact.Fingerprint())
	if err != nil {
		return err
	}

	_, err = dconn.WriteString(r.Self.Fingerprint())
	if err != nil {
		return err
	}

	_, err = dconn.WriteBytes(r.ID[:])
	if err != nil {
		return err
	}

	dconn.Flush()

	remoteConv, err := dconn.ReadString()
	if err != nil {
		return err
	}

	sig, err := dconn.ReadBytes()
	if err != nil {
		return err
	}

	dconn.Close()

	if !contact.Verify(append([]byte(remoteConv), r.ID[:]...), sig) {
		return fmt.Errorf("invalid signature from remote %s", contact.URL())
	}

	remote, err := NewRemoteIdentity(remoteConv)
	if err != nil {
		return err
	}

	log.Printf("Validated %s\n", contact.URL())
	log.Printf("Conversiation ID %s\n", remoteConv)

	r.Peers = append(r.Peers, remote)
	return nil
}

func (r *Room) SendMessage(mtype byte, content []byte) error {
	msg, err := NewMessage(r.Self.Fingerprint(), mtype, content)
	if err != nil {
		return err
	}

	msg.Sign(r.Self.Key)

	r.LogMessage(msg)

	for _, peer := range r.Peers {
		go peer.QueueMessage(msg)
	}

	return nil
}

func (r *Room) RunRemoteMessageQueues(dialer proxy.Dialer, conversationPort int) {
	r.queueTerminate = make(chan bool)
	for _, peer := range r.Peers {
		peer.InitQueue(dialer, conversationPort, r.ID, r.queueTerminate)
		go peer.RunMessageQueue()
	}
}

func (r *Room) PeerByFingerprint(fingerprint string) *RemoteIdentity {
	for _, peer := range r.Peers {
		if peer.Fingerprint() == fingerprint {
			return peer
		}
	}
	return nil
}

func (r *Room) StopQueues() {
	close(r.queueTerminate)
}

func (r *Room) LogMessage(msg *Message) {
	if msg.Type == MTYPE_CMD {
		if msg.GetContent() != nil {
			r.handleCommand(msg)
		}
	}

	r.Messages = append(r.Messages, msg)
}

func (r *Room) handleCommand(msg *Message) {
	cmd := string(msg.GetContent())

	args := strings.Split(cmd, " ")
	switch args[0] {
	case "join":
		if len(args) < 2 {
			log.Printf("Not enough args for command \"%s\"\n", cmd)
			break
		}

		if r.PeerByFingerprint(args[1]) != nil || args[1] == r.Self.Fingerprint() {
			//User already added, or self
			break
		}

		newPeer, err := NewRemoteIdentity(args[1])
		if err != nil {
			log.Println(err.Error())
			break
		}

		r.Peers = append(r.Peers, newPeer)
		log.Printf("New peer %s added to room %s\n", newPeer.Fingerprint(), r.ID)
	case "name_room":
		if len(args) < 2 {
			log.Printf("Not enough args for command \"%s\"\n", cmd)
			break
		}

		r.Name = args[1]
		log.Printf("Room with id %s renamed to %s", r.ID, r.Name)
	case "nick":
		if len(args) < 3 {
			log.Printf("Not enough args for command \"%s\"\n", cmd)
			break
		}

		fingerprint := args[1]
		nickname := args[2]

		if fingerprint == msg.Sender {
			r.Nicks[fingerprint] = nickname
			log.Printf("Set nickname fro %s to %s", fingerprint, nickname)
		} else {
			log.Printf("%s tried to set nickname %s for %s this shouldn't happen!", msg.Sender, nickname, fingerprint)
		}
	default:
		log.Printf("Received invalid command \"%s\"\n", cmd)
	}
}
