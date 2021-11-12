package types

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
)

type Room struct {
	Self     Identity         `json:"self"`
	Peers    []*MessagingPeer `json:"peers"`
	ID       uuid.UUID        `json:"uuid"`
	Name     string           `json:"name"`
	Messages []Message        `json:"messages"`

	connectionManager ConnectionManager
	commandHandler    CommandHandler

	SyncState      SyncMap `json:"lastMessage"`
	msgUpdateMutex sync.Mutex

	ctx  context.Context
	stop context.CancelFunc

	newMessageHook func(uuid.UUID, ...Message)
}

type RoomInfo struct {
	Self   string            `json:"self"`
	Peers  []string          `json:"peers"`
	ID     uuid.UUID         `json:"uuid"`
	Name   string            `json:"name,omitempty"`
	Nicks  map[string]string `json:"nicks,omitempty"`
	Admins map[string]bool   `json:"admins,omitempty"`
}

func NewRoom(ctx context.Context, cManager ConnectionManager, commandHandler CommandHandler, contactIdentities ...Identity) (*Room, error) {
	id, err := NewIdentity(Self, "")
	if err != nil {
		return nil, err
	}

	id.Meta.Admin = true
	room := &Room{
		Self:              id,
		ID:                uuid.New(),
		connectionManager: cManager,
		commandHandler:    commandHandler,
		SyncState:         make(SyncMap),
	}

	room.SetContext(ctx)

	err = room.AddPeers(contactIdentities...)
	if err != nil {
		return nil, err
	}

	return room, nil
}

func (r *Room) SetContext(ctx context.Context) {
	r.ctx, r.stop = context.WithCancel(ctx)
}

func (r *Room) SetConnectionManager(manager ConnectionManager) {
	r.connectionManager = manager
}

func (r *Room) SetCommandHandler(handler CommandHandler) {
	r.commandHandler = handler
}

func (r *Room) SetNewMessageHook(hook func(uuid.UUID, ...Message)) {
	r.newMessageHook = hook
}

/*
AddPeers adds a user to the Room, and if successful syncs the PeerLists.
If not successful returns the error.
*/
func (r *Room) AddPeers(contactIdentities ...Identity) error {
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
		go peer.RunMessageQueue(r.ctx, r)
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
			Data: ConstructCommand([]byte(peer.RIdentity.Fingerprint()), RoomCommandInvite),
		})
	}
}

/*
This function tries to add a user with the contactID to the Room.
This only adds the user, so the user lists are then out of sync.
Call syncPeerLists() to sync them again.
*/
func (r *Room) createPeerViaContactID(contactIdentity Identity) (*MessagingPeer, error) {
	resp, err := r.connectionManager.contactPeer(r, contactIdentity)
	if err != nil {
		return nil, err
	}

	if ok, _ := contactIdentity.Verify(append([]byte(resp.ConvFP), r.ID[:]...), resp.Sig); !ok {
		return nil, fmt.Errorf("invalid signature from contactIdentity %s", contactIdentity.URL())
	}

	switch ok, err := contactIdentity.Verify(append([]byte(resp.ConvFP), r.ID[:]...), resp.Sig); {
	case err != nil:
		return nil, err
	case !ok:
		return nil, fmt.Errorf("invalid signature from contactIdentity %s", contactIdentity.URL())
	}

	peerID, err := NewIdentity(Remote, resp.ConvFP)
	if err != nil {
		return nil, err
	}

	lf := log.Fields{
		"contact-url":     contactIdentity.URL(),
		"conversation-id": resp.ConvFP,
		"peer":            peerID,
		"room":            r.ID.String(),
	}
	log.WithFields(lf).Debug("contact validated and turned into a peer")

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
		go peer.RunMessageQueue(r.ctx, r)
	}
}

func (r *Room) PeerByFingerprint(fingerprint string) (Identity, bool) {
	for _, peer := range r.Peers {
		if peer.RIdentity.Fingerprint() == fingerprint {
			return peer.RIdentity, true
		}
	}
	return Identity{}, false
}

// StopQueues cancels this context and with that all message queues of
// MessagingPeer's in this Room
func (r *Room) StopQueues() {
	log.WithField("room", r.ID.String()).Debug("stopping room")
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
				err := r.commandHandler.HandleCommand(&msg, r)
				if err != nil {
					log.WithError(err).Warn()
				}
			}

			lf := log.Fields{
				"room":    r.ID.String(),
				"message": string(msg.Content.Data),
			}
			log.WithFields(lf).Debug("new message")
			r.Messages = append(r.Messages, msg)
		}
	}

	r.SyncState = newSyncState

	r.msgUpdateMutex.Unlock()

	if (r.newMessageHook != nil) {
		r.newMessageHook(r.ID, msgs...)
	}

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

	info.Nicks[r.Self.Fingerprint()] = r.Self.Meta.Nick
	info.Admins[r.Self.Fingerprint()] = r.Self.Meta.Admin

	for _, peer := range r.Peers {
		info.Peers = append(info.Peers, peer.RIdentity.Fingerprint())
		info.Nicks[peer.RIdentity.Fingerprint()] = peer.RIdentity.Meta.Nick
		info.Admins[peer.RIdentity.Fingerprint()] = peer.RIdentity.Meta.Admin
	}

	return info
}

func (r *Room) removePeer(toRemove string) error {
	for i, peer := range r.Peers {
		if peer.RIdentity.Fingerprint() == toRemove {
			peer.Stop()
			r.Peers = append(r.Peers[:i], r.Peers[i+1:]...)
			return nil
		}
	}

	return peerNotFoundError(toRemove)
}

func (r *Room) findMessagesToSync(remoteSyncTimes SyncMap) []Message {
	msgs := make([]Message, 0)

	for _, msg := range r.Messages {
		if last, ok := remoteSyncTimes[msg.Meta.Sender]; !ok || msg.Meta.Time.After(last) {
			msgs = append(msgs, msg)
		}
	}

	return msgs
}

func (r *Room) syncMsgs(peerRID Identity) error {
	return r.connectionManager.syncMsgs(r, peerRID)
}
