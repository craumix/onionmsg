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
	Self     Identity          `json:"self"`
	Peers    []*MessagingPeer  `json:"peers"`
	ID       uuid.UUID         `json:"uuid"`
	Name     string            `json:"name"`
	Nicks    map[string]string `json:"nicks"`
	Messages []Message         `json:"messages"`

	queueTerminate chan bool
}

type RoomInfo struct {
	Self  string            `json:"self"`
	Peers []string          `json:"peers"`
	ID    uuid.UUID         `json:"uuid"`
	Name  string            `json:"name,omitempty"`
	Nicks map[string]string `json:"nicks,omitempty"`
}

func NewRoom(contactIdentities []RemoteIdentity, dialer proxy.Dialer) (*Room, error) {
	s := NewIdentity()
	id := uuid.New()

	room := &Room{
		Self: s,
		ID:   id,
	}

	errG := new(errgroup.Group)

	for _, contact := range contactIdentities {
		contact := contact
		errG.Go(func() error {
			_, err := room.addUserWithContactID(contact, dialer)
			return err
		})
	}

	if err := errG.Wait(); err != nil {
		return nil, err
	}

	room.RunRemoteMessageQueues(dialer)
	room.syncPeerLists()

	return room, nil
}

/*
AddUser adds a user to the Room, and if successful syncs the PeerLists.
If not successful returns the error.
*/
func (r *Room) AddUser(contact RemoteIdentity, dialer proxy.Dialer) error {
	q, err := r.addUserWithContactID(contact, dialer)
	if err != nil {
		return err
	}

	q.RunMessageQueue(dialer, r)

	r.syncPeerLists()
	return nil
}

/*
Syncs the user list for all peers.
This only adds users, and cant remove users from peers.
*/
func (r *Room) syncPeerLists() {
	for _, peer := range r.Peers {
		r.SendMessage(MTYPE_CMD, []byte("join "+peer.RIdentity.Fingerprint()))
	}
}

/*
This function tries to add a user with the contactID to the room.
This only adds the user, so the user lists are then out of sync.
Call syncPeerLists() to sync them again.
*/
func (r *Room) addUserWithContactID(remote RemoteIdentity, dialer proxy.Dialer) (*MessagingPeer, error) {
	conn, err := dialer.Dial("tcp", remote.URL()+":"+strconv.Itoa(PubContPort))
	if err != nil {
		return nil, err
	}

	dconn := sio.WrapConnection(conn)

	req := &ContactRequest{
		RemoteFP: remote.Fingerprint(),
		LocalFP:  r.Self.Fingerprint(),
		ID:       r.ID,
	}
	_, err = dconn.WriteStruct(req)
	if err != nil {
		return nil, err
	}

	dconn.Flush()

	resp := &ContactResponse{}
	err = dconn.ReadStruct(resp)
	if err != nil {
		return nil, err
	}

	dconn.Close()

	if !remote.Verify(append([]byte(resp.ConvFP), r.ID[:]...), resp.Sig) {
		return nil, fmt.Errorf("invalid signature from remote %s", remote.URL())
	}

	peerID, err := NewRemoteIdentity(resp.ConvFP)
	if err != nil {
		return nil, err
	}

	log.Printf("Validated %s\n", remote.URL())
	log.Printf("Conversiation ID %s\n", resp.ConvFP)

	peer := NewMessagingPeer(peerID)
	r.Peers = append(r.Peers, peer)
	return peer, nil
}

func (r *Room) SendMessage(mtype byte, content []byte) error {
	msg := Message{
		Meta: MessageMeta{
			Sender: r.Self.Fingerprint(),
			Time:   time.Now().UTC(),
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

func (r *Room) RunRemoteMessageQueues(dialer proxy.Dialer) {
	r.queueTerminate = make(chan bool)
	for _, peer := range r.Peers {
		go peer.RunMessageQueue(dialer, r)
	}
}

func (r *Room) PeerByFingerprint(fingerprint string) (RemoteIdentity, bool) {
	for _, peer := range r.Peers {
		if peer.RIdentity.Fingerprint() == fingerprint {
			return peer.RIdentity, true
		}
	}
	return RemoteIdentity{}, false
}

func (r *Room) StopQueues() {
	close(r.queueTerminate)
}

func (r *Room) LogMessage(msg Message) {
	if msg.Meta.Type == MTYPE_CMD {
		r.handleCommand(msg)
	}

	r.Messages = append(r.Messages, msg)
}

func (r *Room) handleCommand(msg Message) {
	cmd := string(msg.Content)

	args := strings.Split(cmd, " ")
	switch args[0] {
	case "join":
		if len(args) < 2 {
			log.Printf("Not enough args for command \"%s\"\n", cmd)
			break
		}

		if _, ok := r.PeerByFingerprint(args[1]); ok || args[1] == r.Self.Fingerprint() {
			//User already added, or self
			break
		}

		peerID, err := NewRemoteIdentity(args[1])
		if err != nil {
			log.Println(err.Error())
			break
		}

		newPeer := NewMessagingPeer(peerID)
		r.Peers = append(r.Peers, newPeer)

		//TODO start queue, how get proxy here? Maybe just make global.

		log.Printf("New peer %s added to room %s\n", newPeer.RIdentity.Fingerprint(), r.ID)
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

//Info returns a struct with most information about this room
func (r *Room) Info() *RoomInfo {
	info := &RoomInfo{
		Self:  r.Self.Fingerprint(),
		ID:    r.ID,
		Name:  r.Name,
		Nicks: r.Nicks,
	}

	for _, p := range r.Peers {
		info.Peers = append(info.Peers, p.RIdentity.Fingerprint())
	}

	return info
}
