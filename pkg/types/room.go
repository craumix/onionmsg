package types

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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

func NewRoom(contactIdentities []RemoteIdentity) (*Room, error) {
	room := &Room{
		Self: NewIdentity(),
		ID:   uuid.New(),
	}

	err := room.addUsers(contactIdentities)
	if err != nil {
		return nil, err
	}

	room.RunRemoteMessageQueues()
	room.syncPeerLists()

	return room, nil
}

func (self *Room) addUsers(contactIdentities []RemoteIdentity) error {
	errG := new(errgroup.Group)

	for _, contact := range contactIdentities {
		contact := contact
		errG.Go(func() error {
			_, err := self.addUserWithContactID(contact)
			return err
		})
	}

	return errG.Wait()
}

/*
AddUser adds a user to the Room, and if successful syncs the PeerLists.
If not successful returns the error.
*/
func (r *Room) AddUser(contact RemoteIdentity) error {
	q, err := r.addUserWithContactID(contact)
	if err != nil {
		return err
	}

	err = q.RunMessageQueue(r)
	if err != nil {
		return err
	}

	r.syncPeerLists()
	return nil
}

/*
Syncs the user list for all peers.
This only adds users, and cant remove users from peers.
*/
func (r *Room) syncPeerLists() {
	for _, peer := range r.Peers {
		r.SendMessage(MessageTypeCmd, []byte("join "+peer.RIdentity.Fingerprint()))
	}
}

/*
This function tries to add a user with the contactID to the room.
This only adds the user, so the user lists are then out of sync.
Call syncPeerLists() to sync them again.
*/
func (r *Room) addUserWithContactID(remote RemoteIdentity) (*MessagingPeer, error) {
	dconn, err := sio.DialDataConn("tcp", remote.URL()+":"+strconv.Itoa(PubContPort))
	if err != nil {
		return nil, err
	}
	defer dconn.Close()

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

func (r *Room) SendMessage(mtype MessageType, content []byte) error {
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

func (r *Room) RunRemoteMessageQueues() {
	r.queueTerminate = make(chan bool)
	for _, peer := range r.Peers {
		go peer.RunMessageQueue(r)
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
	if msg.Meta.Type == MessageTypeCmd {
		r.handleCommand(msg)
	}

	r.Messages = append(r.Messages, msg)
}

// Info returns a struct with most information about this room
func (self *Room) Info() *RoomInfo {
	info := &RoomInfo{
		Self:  self.Self.Fingerprint(),
		ID:    self.ID,
		Name:  self.Name,
		Nicks: self.Nicks,
	}

	for _, p := range self.Peers {
		info.Peers = append(info.Peers, p.RIdentity.Fingerprint())
	}

	return info
}

func (self *Room) handleCommand(msg Message) {
	cmd := string(msg.Content)

	args := strings.Split(cmd, " ")
	switch args[0] {
	case "join":
		self.handleJoin(args)
		break
	case "name_room":
		self.handleNameRoom(args)
		break
	case "nick":
		self.handleNick(args, msg.Meta.Sender)
		break
	default:
		log.Printf("Received invalid command \"%s\"\n", cmd)
	}
}

func (self *Room) handleJoin(args []string) {
	if !enoughArgs(args, 2) {
		return
	}

	if _, ok := self.PeerByFingerprint(args[1]); ok || args[1] == self.Self.Fingerprint() {
		//User already added, or self
		return
	}

	peerID, err := NewRemoteIdentity(args[1])
	if err != nil {
		log.Println(err.Error())
		return
	}

	newPeer := NewMessagingPeer(peerID)
	self.Peers = append(self.Peers, newPeer)

	go newPeer.RunMessageQueue(self)

	log.Printf("New peer %s added to room %s\n", newPeer.RIdentity.Fingerprint(), self.ID)
}

func (self *Room) handleNameRoom(args []string) {
	if !enoughArgs(args, 2) {
		return
	}

	self.Name = args[1]
	log.Printf("Room with id %s renamed to %s", self.ID, self.Name)
}

func (self Room) handleNick(args []string, sender string) {
	if !enoughArgs(args, 2) {
		return
	}

	nickname := args[1]

	self.Nicks[sender] = nickname
	log.Printf("Set nickname fro %s to %s", sender, nickname)
}

func enoughArgs(args []string, needed int) bool {
	if len(args) < needed {
		log.Printf("Not enough args for command \"%s\"\n", strings.Join(args, " "))
		return false
	}
	return true
}
