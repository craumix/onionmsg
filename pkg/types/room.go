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

func (r *Room) SetContext(ctx context.Context) error {
	if r.ctx == nil {
		r.ctx, r.stop = context.WithCancel(ctx)
		return nil
	}
	return fmt.Errorf("%s already has a context", r.ID.String())
}

func (r *Room) addUsers(contactIdentities []RemoteIdentity) error {
	errG := new(errgroup.Group)

	for _, contact := range contactIdentities {
		contact := contact
		errG.Go(func() error {
			_, err := r.addUserWithContactID(contact)
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
	peer, err := r.addUserWithContactID(contact)
	if err != nil {
		return err
	}

	go peer.RunMessageQueue(r.ctx, r)

	r.syncPeerLists()
	return nil
}

/*
Syncs the user list for all peers.
This only adds users, and can't remove users from peers.
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
	dataConn, err := sio.DialDataConn("tcp", remote.URL()+":"+strconv.Itoa(PubContPort))
	if err != nil {
		return nil, err
	}
	defer dataConn.Close()

	req := &ContactRequest{
		RemoteFP: remote.Fingerprint(),
		LocalFP:  r.Self.Fingerprint(),
		ID:       r.ID,
	}
	_, err = dataConn.WriteStruct(req)
	if err != nil {
		return nil, err
	}

	dataConn.Flush()

	resp := &ContactResponse{}
	err = dataConn.ReadStruct(resp)
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

func (r *Room) SendMessage(msgType MessageType, content []byte) error {
	msg := Message{
		Meta: MessageMeta{
			Sender: r.Self.Fingerprint(),
			Time:   time.Now().UTC(),
			Type:   msgType,
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
	for _, peer := range r.Peers {
		go peer.RunMessageQueue(r.ctx, r)
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
	log.Printf("Stopping Room %s", r.ID.String())
	r.stop()
}

func (r *Room) LogMessage(msg Message) {
	if msg.Meta.Type == MessageTypeCmd {
		r.handleCommand(msg)
	}

	r.Messages = append(r.Messages, msg)
}

// Info returns a struct with most information about this room
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

func (r *Room) handleCommand(msg Message) {
	cmd := string(msg.Content)

	args := strings.Split(cmd, " ")
	switch args[0] {
	case "join":
		r.handleJoin(args)
	case "name_room":
		r.handleNameRoom(args)
	case "nick":
		r.handleNick(args, msg.Meta.Sender)
	default:
		log.Printf("Received invalid command \"%s\"\n", cmd)
	}
}

func (r *Room) handleJoin(args []string) {
	if !enoughArgs(args, 2) {
		return
	}

	if _, ok := r.PeerByFingerprint(args[1]); ok || args[1] == r.Self.Fingerprint() {
		//User already added, or self
		return
	}

	peerID, err := NewRemoteIdentity(args[1])
	if err != nil {
		log.Println(err.Error())
		return
	}

	newPeer := NewMessagingPeer(peerID)
	r.Peers = append(r.Peers, newPeer)

	go newPeer.RunMessageQueue(r.ctx, r)

	log.Printf("New peer %s added to room %s\n", newPeer.RIdentity.Fingerprint(), r.ID)
}

func (r *Room) handleNameRoom(args []string) {
	if !enoughArgs(args, 2) {
		return
	}

	r.Name = args[1]
	log.Printf("Room with id %s renamed to %s", r.ID, r.Name)
}

func (r Room) handleNick(args []string, sender string) {
	if !enoughArgs(args, 2) {
		return
	}

	nickname := args[1]

	r.Nicks[sender] = nickname
	log.Printf("Set nickname fro %s to %s", sender, nickname)
}

func enoughArgs(args []string, needed int) bool {
	if len(args) < needed {
		log.Printf("Not enough args for command \"%s\"\n", strings.Join(args, " "))
		return false
	}
	return true
}
