package types

import (
	"context"
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

	ctx  context.Context
	stop context.CancelFunc
}

type RoomInfo struct {
	Self  string            `json:"self"`
	Peers []string          `json:"peers"`
	ID    uuid.UUID         `json:"uuid"`
	Name  string            `json:"name,omitempty"`
	Nicks map[string]string `json:"nicks,omitempty"`
}

func NewRoom(ctx context.Context, contactIdentities []RemoteIdentity) (*Room, error) {
	room := &Room{
		Self: NewIdentity(),
		ID:   uuid.New(),
	}

	err := room.SetContext(ctx)
	if err != nil {
		return nil, err
	}

	err = room.addUsers(contactIdentities)
	if err != nil {
		return nil, err
	}

	room.RunRemoteMessageQueues()
	room.syncPeerLists()

	return room, nil
}

func (self *Room) SetContext(ctx context.Context) error {
	if self.ctx == nil {
		self.ctx, self.stop = context.WithCancel(ctx)
		return nil
	}
	return fmt.Errorf("%s already has a context", self.ID.String())
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
func (self *Room) AddUser(contact RemoteIdentity) error {
	peer, err := self.addUserWithContactID(contact)
	if err != nil {
		return err
	}

	go peer.RunMessageQueue(self.ctx, self)

	self.syncPeerLists()
	return nil
}

/*
Syncs the user list for all peers.
This only adds users, and can't remove users from peers.
*/
func (self *Room) syncPeerLists() {
	for _, peer := range self.Peers {
		self.SendMessage(MessageTypeCmd, []byte("join "+peer.RIdentity.Fingerprint()))
	}
}

/*
This function tries to add a user with the contactID to the room.
This only adds the user, so the user lists are then out of sync.
Call syncPeerLists() to sync them again.
*/
func (self *Room) addUserWithContactID(remote RemoteIdentity) (*MessagingPeer, error) {
	dconn, err := sio.DialDataConn("tcp", remote.URL()+":"+strconv.Itoa(PubContPort))
	if err != nil {
		return nil, err
	}
	defer dconn.Close()

	req := &ContactRequest{
		RemoteFP: remote.Fingerprint(),
		LocalFP:  self.Self.Fingerprint(),
		ID:       self.ID,
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

	if !remote.Verify(append([]byte(resp.ConvFP), self.ID[:]...), resp.Sig) {
		return nil, fmt.Errorf("invalid signature from remote %s", remote.URL())
	}

	peerID, err := NewRemoteIdentity(resp.ConvFP)
	if err != nil {
		return nil, err
	}

	log.Printf("Validated %s\n", remote.URL())
	log.Printf("Conversiation ID %s\n", resp.ConvFP)

	peer := NewMessagingPeer(peerID)
	self.Peers = append(self.Peers, peer)
	return peer, nil
}

func (self *Room) SendMessage(mtype MessageType, content []byte) error {
	msg := Message{
		Meta: MessageMeta{
			Sender: self.Self.Fingerprint(),
			Time:   time.Now().UTC(),
			Type:   mtype,
		},
		Content: content,
	}

	self.LogMessage(msg)

	for _, peer := range self.Peers {
		go peer.QueueMessage(msg)
	}

	return nil
}

func (self *Room) RunRemoteMessageQueues() {
	for _, peer := range self.Peers {
		go peer.RunMessageQueue(self.ctx, self)
	}
}

func (self *Room) PeerByFingerprint(fingerprint string) (RemoteIdentity, bool) {
	for _, peer := range self.Peers {
		if peer.RIdentity.Fingerprint() == fingerprint {
			return peer.RIdentity, true
		}
	}
	return RemoteIdentity{}, false
}

func (self *Room) StopQueues() {
	log.Printf("Stopping Room %s", self.ID.String())
	self.stop()
}

func (self *Room) LogMessage(msg Message) {
	if msg.Meta.Type == MessageTypeCmd {
		self.handleCommand(msg)
	}

	self.Messages = append(self.Messages, msg)
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
	case "name_room":
		self.handleNameRoom(args)
	case "nick":
		self.handleNick(args, msg.Meta.Sender)
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

	go newPeer.RunMessageQueue(self.ctx, self)

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
