package types

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/craumix/onionmsg/pkg/sio/connection"

	"github.com/google/uuid"
)

type Room struct {
	Self     Identity         `json:"self"`
	Peers    []*MessagingPeer `json:"peers"`
	ID       uuid.UUID        `json:"uuid"`
	Name     string           `json:"name"`
	Messages []Message        `json:"messages"`

	SyncState      SyncMap `json:"lastMessage"`
	msgUpdateMutex sync.Mutex

	Ctx  context.Context `json:"-"`
	stop context.CancelFunc
}

type RoomInfo struct {
	Self   string            `json:"self"`
	Peers  []string          `json:"peers"`
	ID     uuid.UUID         `json:"uuid"`
	Name   string            `json:"name,omitempty"`
	Nicks  map[string]string `json:"nicks,omitempty"`
	Admins map[string]bool   `json:"admins,omitempty"`
}

func NewRoom(ctx context.Context, contactIdentities ...RemoteIdentity) (*Room, error) {
	room := &Room{
		Self:      NewIdentity(),
		ID:        uuid.New(),
		SyncState: make(SyncMap),
	}
	room.Self.Meta.Admin = true

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
	newSyncState := CopySyncMap(r.SyncState)

	r.msgUpdateMutex.Lock()

	//Usually all messages that reach this point should be new to us,
	//the if-statement is more of a failsafe
	for _, msg := range msgs {
		if last, ok := r.SyncState[msg.Meta.Sender]; !ok || msg.Meta.Time.After(last) {
			newSyncState[msg.Meta.Sender] = msg.Meta.Time

			if msg.Content.Type == ContentTypeCmd {
				err := HandleCommand(&msg, r)
				if err != nil {
					log.Print(err.Error())
				}
			}

			log.Printf("New message for room %s: %s", r.ID, msg.Content.Data)
			r.Messages = append(r.Messages, msg)
		}
	}

	r.SyncState = newSyncState

	r.msgUpdateMutex.Unlock()

	return nil
}

func (r *Room) isSelf(fingerprint string) bool {
	return fingerprint == r.Self.Fingerprint()
}

// Info returns a struct with useful information about this Room
func (r *Room) Info() *RoomInfo {
	info := &RoomInfo{
		Self:   r.Self.Fingerprint(),
		ID:     r.ID,
		Name:   r.Name,
		Nicks:  map[string]string{},
		Admins: map[string]bool{},
	}

	for _, peer := range r.Peers {
		info.Peers = append(info.Peers, peer.RIdentity.Fingerprint())
		info.Nicks[peer.RIdentity.Fingerprint()] = peer.RIdentity.Meta.Nick
		info.Admins[peer.RIdentity.Fingerprint()] = peer.RIdentity.Meta.Admin
	}

	return info
}
