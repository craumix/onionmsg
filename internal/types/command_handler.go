package types

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Command string

const (
	RoomCommandInvite     Command = "invite"
	RoomCommandNameRoom   Command = "name_room"
	RoomCommandNick       Command = "nick"
	RoomCommandPromote    Command = "promote"
	RoomCommandRemovePeer Command = "remove_peer"

	// RoomCommandAccept is essentially a No-Op,
	//and is mainly used for indication in frontends
	RoomCommandAccept Command = "accept"

	CommandDelimiter = " "
)

type CommandHandler struct {
	commandCallbacks map[Command]func(Command, *Message, *Room) error
}

func NewCommandHandler(commandCallbacks map[Command]func(Command, *Message, *Room) error) CommandHandler {
	return CommandHandler{
		commandCallbacks: commandCallbacks,
	}
}

func GetDefaultCommandHandler() CommandHandler {
	ch := CommandHandler{
		commandCallbacks: make(map[Command]func(Command, *Message, *Room) error),
	}

	ch.RegisterCommand(RoomCommandInvite, inviteCallback)

	ch.RegisterCommand(RoomCommandNameRoom, nameRoomCallback)

	ch.RegisterCommand(RoomCommandNick, nickCallback)

	ch.RegisterCommand(RoomCommandPromote, promoteCallback)

	ch.RegisterCommand(RoomCommandRemovePeer, removePeerCallback)

	ch.RegisterCommand(RoomCommandAccept, nil)

	return ch
}

func (ch *CommandHandler) RegisterCommand(command Command, callback func(Command, *Message, *Room) error) error {
	if _, found := ch.commandCallbacks[command]; found {
		return fmt.Errorf("command %s is already registered", command)
	}
	ch.commandCallbacks[command] = callback
	return nil
}

func (ch CommandHandler) HandleCommand(message *Message, room *Room) error {
	hasCommand, command := message.isCommand()
	if !hasCommand {
		return fmt.Errorf("message isn't a command")
	}
	if _, found := ch.commandCallbacks[Command(command)]; !found {
		return fmt.Errorf("command %s is not registered", command)
	}

	handleFunc := ch.commandCallbacks[Command(command)]
	if handleFunc != nil {
		return handleFunc(Command(command), message, room)
	}
	return nil
}

func inviteCallback(command Command, message *Message, room *Room) error {
	args, err := parseCommand(message, command, RoomCommandInvite, 2)
	if err != nil {
		return err
	}

	if _, found := room.PeerByFingerprint(args[1]); found || args[1] == room.Self.Fingerprint() {
		return fmt.Errorf("user %s already added, or self", args[1])
	}

	peerID, err := NewIdentity(Remote, args[1])
	if err != nil {
		return err
	}

	newPeer := NewMessagingPeer(peerID)
	room.Peers = append(room.Peers, newPeer)

	go newPeer.RunMessageQueue(room.ctx, room)

	lf := log.Fields{
		"peer": newPeer.RIdentity.Fingerprint(),
		"room": room.ID,
	}
	log.WithFields(lf).Debug("new peer added to room")
	return nil
}

func nameRoomCallback(command Command, message *Message, room *Room) error {
	args, err := parseCommand(message, command, RoomCommandNameRoom, 2)
	if err != nil {
		return err
	}

	room.Name = args[1]
	log.Debugf("Room with id %s renamed to %s", room.ID, room.Name)

	return nil
}

func nickCallback(command Command, message *Message, room *Room) error {
	args, err := parseCommand(message, command, RoomCommandNick, 2)
	if err != nil {
		return err
	}

	sender, err := getSender(message, room, false)
	if err != nil {
		return err
	}

	nickname := args[1]
	sender.Meta.Nick = nickname
	log.Debugf("Set nickname for %s to %s", sender.Fingerprint(), nickname)

	return nil
}

func promoteCallback(command Command, message *Message, room *Room) error {
	args, err := parseCommand(message, command, RoomCommandPromote, 2)
	if err != nil {
		return err
	}

	_, err = getSender(message, room, true)
	if err != nil {
		return err
	}

	toPromote, found := room.PeerByFingerprint(args[1])
	switch {
	case found:
		toPromote.Meta.Admin = true
	case room.isSelf(args[1]):
		room.Self.Meta.Admin = true
	default:
		return peerNotFoundError(args[1])
	}

	return nil
}

func removePeerCallback(command Command, message *Message, room *Room) error {
	args, err := parseCommand(message, command, RoomCommandRemovePeer, 2)
	if err != nil {
		return err
	}

	_, err = getSender(message, room, true)
	if err != nil {
		return err
	}

	return room.removePeer(args[1])
}

func getSender(msg *Message, r *Room, shouldBeAdmin bool) (Identity, error) {
	sender, found := r.PeerByFingerprint(msg.Meta.Sender)
	if !found {
		if r.Self.Fingerprint() != msg.Meta.Sender {
			return Identity{}, peerNotFoundError(msg.Meta.Sender)
		}
		sender = r.Self
	}

	if shouldBeAdmin && !sender.Meta.Admin {
		return Identity{}, peerNotAdminError(msg.Meta.Sender)
	}

	return sender, nil
}

func parseCommand(message *Message, actualCommand, expectedCommand Command, expectedArgs int) ([]string, error) {
	if actualCommand != expectedCommand {
		return nil, fmt.Errorf("%s is the wrong command", actualCommand)
	}

	args := strings.Split(string(message.Content.Data), CommandDelimiter)
	if !enoughArgs(args, expectedArgs) {
		return nil, fmt.Errorf("%s doesn't have enough arguments", actualCommand)
	}

	return strings.Split(string(message.Content.Data), CommandDelimiter), nil
}

func enoughArgs(args []string, needed int) bool {
	if len(args) < needed {
		log.Warnf("Not enough args for command \"%s\"\n", strings.Join(args, " "))
		return false
	}
	return true
}

func peerNotFoundError(peer string) error {
	return fmt.Errorf("peer %s not found", peer)
}

func peerNotAdminError(peer string) error {
	return fmt.Errorf("peer %s is not an admin", peer)
}

func ConstructCommand(message []byte, command Command) []byte {
	if command == "" {
		return message
	} else if len(message) == 0 {
		return []byte(command)
	}

	return []byte(string(command) + CommandDelimiter + string(message))
}

func (ch *CommandHandler) CleanCallbacks() {
	ch.commandCallbacks = map[Command]func(Command, *Message, *Room) error{}
}
