package types

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/proxy"
	"golang.org/x/sync/errgroup"

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

	errG := new(errgroup.Group)

	for _, contact := range contactIdentities {
		contact := contact
		errG.Go(func() error {
			return room.addUserWithContactID(contact, dialer, contactPort)
		})
	}

	if err := errG.Wait(); err != nil {
		return nil, err
	}

	room.syncPeerLists()

	return room, nil
}

/*
AddUser adds a user to the Room, and if successful syncs the PeerLists.
If not successful returns the error.
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
func (r *Room) addUserWithContactID(remote *RemoteIdentity, dialer proxy.Dialer, contactPort int) error {
	conn, err := dialer.Dial("tcp", remote.URL()+":"+strconv.Itoa(contactPort))
	if err != nil {
		return err
	}

	dconn := sio.NewDataIO(conn)

	req := &ContactRequest{
		RemoteFP: remote.Fingerprint(),
		LocalFP:  r.Self.Fingerprint(),
		ID:       r.ID,
	}
	_, err = dconn.WriteStruct(req)
	if err != nil {
		return err
	}

	dconn.Flush()

	resp := &ContactResponse{}
	err = dconn.ReadStruct(resp)
	if err != nil {
		return err
	}

	dconn.Close()

	if !remote.Verify(append([]byte(resp.ConvFP), r.ID[:]...), resp.Sig) {
		return fmt.Errorf("invalid signature from remote %s", remote.URL())
	}

	peer, err := NewRemoteIdentity(resp.ConvFP)
	if err != nil {
		return err
	}

	log.Printf("Validated %s\n", remote.URL())
	log.Printf("Conversiation ID %s\n", resp.ConvFP)

	r.Peers = append(r.Peers, peer)
	return nil
}

func (r *Room) SendMessage(mtype byte, content []byte) error {
	msg := &Message{
		Meta: MessageMeta{
			Sender: r.Self.Fingerprint(),
			Time:   time.Now(),
			Type:   mtype,
		},
		Content: content,
	}

	r.LogMessage(msg)

	for _, peer := range r.Peers {
		go peer.QueueMessage(msg)
	}

	return nil
}

func (r *Room) RunRemoteMessageQueues(dialer proxy.Dialer, conversationPort int) {
	r.queueTerminate = make(chan bool)
	for _, peer := range r.Peers {
		peer.InitQueue(r.Self, dialer, conversationPort, r.ID, r.queueTerminate)
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
	if msg.Meta.Type == MTYPE_CMD {
		r.handleCommand(msg)
	}

	r.Messages = append(r.Messages, msg)
}

func (r *Room) handleCommand(msg *Message) {
	cmd := string(msg.Content)

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
		if len(args) < 2 {
			log.Printf("Not enough args for command \"%s\"\n", cmd)
			break
		}
		nickname := args[1]

		r.Nicks[msg.Meta.Sender] = nickname
		log.Printf("Set nickname fro %s to %s", msg.Meta.Sender, nickname)
	default:
		log.Printf("Received invalid command \"%s\"\n", cmd)
	}
}
