package types

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/craumix/onionmsg/pkg/sio/connection"

	"github.com/google/uuid"
)

type RoomCommand string

const (
	RoomCommandJoin     RoomCommand = "join"
	RoomCommandNameRoom RoomCommand = "name_room"
	RoomCommandNick     RoomCommand = "nick"
)

type Room struct {
	Self     Identity         `json:"self"`
	Peers    []*MessagingPeer `json:"peers"`
	ID       uuid.UUID        `json:"uuid"`
	Name     string           `json:"name"`
	Messages []Message        `json:"messages"`

	SyncTimes   map[string]time.Time `json:"lastMessage"`
	msgUpdateMutex sync.Mutex

	Ctx  context.Context `json:"-"`
	stop context.CancelFunc
}

type RoomInfo struct {
	Self  string            `json:"self"`
	Peers []string          `json:"peers"`
	ID    uuid.UUID         `json:"uuid"`
	Name  string            `json:"name,omitempty"`
	Nicks map[string]string `json:"nicks,omitempty"`
}

func NewRoom(ctx context.Context, contactIdentities ...RemoteIdentity) (*Room, error) {
	room := &Room{
		Self: NewIdentity(),
		ID:   uuid.New(),
		SyncTimes: make(map[string]time.Time),
	}

	err := room.SetContext(ctx)
	if err != nil {
		return nil, err
	}

	err = room.AddPeers(contactIdentities...)
	if err != nil {
		return nil, err
	}

	return room, nil
}

func (r *Room) SetContext(ctx context.Context) error {
	if r.Ctx == nil {
		r.Ctx, r.stop = context.WithCancel(ctx)
		return nil
	}
	return fmt.Errorf("%s already has a context", r.ID.String())
}

/*
AddPeers adds a user to the Room, and if successful syncs the PeerLists.
If not successful returns the error.
*/
func (r *Room) AddPeers(contactIdentities ...RemoteIdentity) error {
	var newPeers []*MessagingPeer
	for _, identity := range contactIdentities {
		newPeer, err := r.createPeerViaContactID(identity)
		if err != nil {
			return err
		}
		newPeers = append(newPeers, newPeer)
	}

	r.Peers = append(r.Peers, newPeers...)

	for _, peer := range newPeers {
		go peer.RunMessageQueue(r.Ctx, r)
	}

	r.syncPeerLists()

	return nil
}

/*
Syncs the user list for all peers.
This only adds users, and can't remove users from peers.
*/
func (r *Room) syncPeerLists() {
	for _, peer := range r.Peers {
		r.SendMessageToAllPeers(MessageContent{
			Type: ContentTypeCmd,
			//TODO make it easier to create command messages
			Data: []byte(string(RoomCommandJoin) + " " + peer.RIdentity.Fingerprint()),
		})
	}
}

/*
This function tries to add a user with the contactID to the Room.
This only adds the user, so the user lists are then out of sync.
Call syncPeerLists() to sync them again.
*/
func (r *Room) createPeerViaContactID(contactIdentity RemoteIdentity) (*MessagingPeer, error) {
	dataConn, err := connection.GetConnFunc("tcp", contactIdentity.URL()+":"+strconv.Itoa(PubContPort))
	if err != nil {
		return nil, err
	}
	defer dataConn.Close()

	req := &ContactRequest{
		RemoteFP: contactIdentity.Fingerprint(),
		LocalFP:  r.Self.Fingerprint(),
		ID:       r.ID,
	}
	_, err = dataConn.WriteStruct(req, false)
	if err != nil {
		return nil, err
	}

	dataConn.Flush()

	resp := &ContactResponse{}
	err = dataConn.ReadStruct(resp, false)
	if err != nil {
		return nil, err
	}

	if !contactIdentity.Verify(append([]byte(resp.ConvFP), r.ID[:]...), resp.Sig) {
		return nil, fmt.Errorf("invalid signature from contactIdentity %s", contactIdentity.URL())
	}

	peerID, err := NewRemoteIdentity(resp.ConvFP)
	if err != nil {
		return nil, err
	}

	log.Printf("Validated %s\n", contactIdentity.URL())
	log.Printf("Conversiation ID %s\n", resp.ConvFP)

	peer := NewMessagingPeer(peerID)
	return peer, nil
}

func (r *Room) SendMessageToAllPeers(content MessageContent) {
	msg := NewMessage(content, r.Self)

	r.PushMessages(msg)

	for _, peer := range r.Peers {
		peer.BumpQueue()
	}
}

func (r *Room) RunMessageQueueForAllPeers() {
	for _, peer := range r.Peers {
		go peer.RunMessageQueue(r.Ctx, r)
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

// StopQueues cancels this context and with that all message queues of
// MessagingPeer's in this Room
func (r *Room) StopQueues() {
	log.Printf("Stopping Room %s", r.ID.String())
	r.stop()
}

func (r *Room) PushMessages(msgs ...Message) error {
	//TODO could be done without sorting, but by duplicating the map and updating the copy and then
	//replacing the original, but this would require a map deepcopy func which I am to lazy for atm
	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Meta.Time.Before(msgs[j].Meta.Time)
	})

	r.msgUpdateMutex.Lock()

	//Usually all messages that reach this point should be new to us,
	//the if-statement is more of a failsafe
	for _, msg := range msgs {
		if last, ok := r.SyncTimes[msg.Meta.Sender]; !ok || msg.Meta.Time.After(last) {
			r.SyncTimes[msg.Meta.Sender] = msg.Meta.Time

			if msg.Content.Type == ContentTypeCmd {
				r.handleCommand(msg)
			}
			
			log.Printf("New message for room %s: %s", r.ID, msg.Content.Data)
			r.Messages = append(r.Messages, msg)
		}
	}

	r.msgUpdateMutex.Unlock()

	return nil
}

// Info returns a struct with useful information about this Room
func (r *Room) Info() *RoomInfo {
	info := &RoomInfo{
		Self:  r.Self.Fingerprint(),
		ID:    r.ID,
		Name:  r.Name,
		Nicks: map[string]string{},
	}

	for _, peer := range r.Peers {
		info.Peers = append(info.Peers, peer.RIdentity.Fingerprint())
		info.Nicks[peer.RIdentity.Fingerprint()] = peer.RIdentity.Nick
	}

	return info
}

func (r *Room) handleCommand(msg Message) {
	cmd := string(msg.Content.Data)

	args := strings.Split(cmd, " ")
	switch args[0] {
	case string(RoomCommandJoin):
		r.handleJoin(args)
	case string(RoomCommandNameRoom):
		r.handleNameRoom(args)
	case string(RoomCommandNick):
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

	go newPeer.RunMessageQueue(r.Ctx, r)

	log.Printf("New peer %s added to Room %s\n", newPeer.RIdentity.Fingerprint(), r.ID)
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

	identity, found := r.PeerByFingerprint(sender)
	if found {
		nickname := args[1]
		identity.Nick = nickname
		log.Printf("Set nickname for %s to %s", sender, nickname)
	} else {
		log.Printf("Peer %s not found", sender)
	}

}

func enoughArgs(args []string, needed int) bool {
	if len(args) < needed {
		log.Printf("Not enough args for command \"%s\"\n", strings.Join(args, " "))
		return false
	}
	return true
}
