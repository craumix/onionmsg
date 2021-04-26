package types

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/proxy"

	"github.com/craumix/onionmsg/internal/sio"
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

	for _, c := range contactIdentities {
		conn, err := dialer.Dial("tcp", c.URL()+":"+strconv.Itoa(contactPort))
		if err != nil {
			return nil, err
		}

		dconn := sio.NewDataIO(conn)

		_, err = dconn.WriteString(c.Fingerprint())
		if err != nil {
			return nil, err
		}

		_, err = dconn.WriteString(s.Fingerprint())
		if err != nil {
			return nil, err
		}

		_, err = dconn.WriteBytes(id[:])
		if err != nil {
			return nil, err
		}

		dconn.Flush()

		remoteConv, err := dconn.ReadString()
		if err != nil {
			return nil, err
		}

		sig, err := dconn.ReadBytes()
		if err != nil {
			return nil, err
		}

		dconn.Close()

		if !c.Verify(append([]byte(remoteConv), id[:]...), sig) {
			return nil, fmt.Errorf("invalid signature from remote %s", c.URL())
		}

		r, err := NewRemoteIdentity(remoteConv)
		if err != nil {
			return nil, err
		}

		log.Printf("Validated %s\n", c.URL())
		log.Printf("Conversiation ID %s\n", remoteConv)

		peers = append(peers, r)
	}

	room := &Room{
		Self:     s,
		Peers:    peers,
		ID:       id,
		Messages: make([]*Message, 0),
		Nicks: make(map[string]string),
	}

	for _, peer := range peers {
		room.SendMessage(MTYPE_CMD, []byte("join " + peer.Fingerprint()))
	}

	return room, nil
}

func (r *Room) SendMessage(mtype byte, content []byte) {
	msg := &Message{
		Sender:  r.Self.Fingerprint(),
		Time:    time.Now(),
		Type:    mtype,
		Content: content,
	}
	msg.Sign(r.Self.Key)

	r.LogMessage(msg)

	for _, peer := range r.Peers {
		go peer.QueueMessage(msg)
	}
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
		if msg.Content != nil {
			r.handleCommand(msg)
		}
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
		if len(args) < 3 {
			log.Printf("Not enough args for command \"%s\"\n", cmd)
			break
		}

		fingerprint := args[1]
		nickname := args[2]

		if(fingerprint == msg.Sender) {
			r.Nicks[fingerprint] = nickname
			log.Printf("Set nickname fro %s to %s", fingerprint, nickname)
		}else {
			log.Printf("%s tried to set nickname %s for %s this shouldn't happen!", msg.Sender, nickname, fingerprint)
		}		
	default:
		log.Printf("Received invalid command \"%s\"\n", cmd)
	}
}
