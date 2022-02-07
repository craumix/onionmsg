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

	ch.RegisterCommand(RoomCommandAccept, noop)

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

	return ch.commandCallbacks[Command(command)](Command(command), message, room)
}

func inviteCallback(command Command, message *Message, room *Room) error {
	args, err := parseCommand(message, command, RoomCommandInvite, 2)
	if err != nil {
		return err
	}

	if _, found := room.PeerByFingerprint(Fingerprint(args[1])); found || args[1] == string(room.Self.Fingerprint()) {
		return fmt.Errorf("user %s already added, or self", args[1])
	}

	peerID, err := NewRemoteIdentity(Fingerprint(args[1]))
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

	sender, self, err := getSender(message, room, false)

	nickname := args[1]
	debugMsg := ""
	switch {
	case err != nil:
		return err
	case self != nil:
		room.Self.SetNick(nickname)
		debugMsg = string(room.Self.Fingerprint())
	case sender != nil:
		sender.SetNick(nickname)
		debugMsg = string(sender.Fingerprint())
	}

	log.Debugf("Set nickname for %s to %s", debugMsg, nickname)

	return nil
}

func promoteCallback(command Command, message *Message, room *Room) error {
	args, err := parseCommand(message, command, RoomCommandPromote, 2)
	if err != nil {
		return err
	}

	_, _, err = getSender(message, room, true)
	if err != nil {
		return err
	}

	toPromote, found := room.PeerByFingerprint(Fingerprint(args[1]))
	switch {
	case found:
		toPromote.SetAdmin(true)
	case room.isSelf(Fingerprint(args[1])):
		room.Self.SetAdmin(true)
	default:
		return peerNotFoundError(Fingerprint(args[1]))
	}

	return nil
}

func removePeerCallback(command Command, message *Message, room *Room) error {
	args, err := parseCommand(message, command, RoomCommandRemovePeer, 2)
	if err != nil {
		return err
	}

	_, _, err = getSender(message, room, true)
	if err != nil {
		return err
	}

	return room.removePeer(Fingerprint(args[1]))
}

func noop(_ Command, _ *Message, _ *Room) error {
	return nil
}

func getSender(msg *Message, r *Room, shouldBeAdmin bool) (*RemoteIdentity, *SelfIdentity, error) {
	isSelf := false
	sender, found := r.PeerByFingerprint(Fingerprint(msg.Meta.Sender))
	if !found {
		if r.Self.Fingerprint() != msg.Meta.Sender {
			return nil, nil, peerNotFoundError(Fingerprint(msg.Meta.Sender))
		}
		isSelf = true
	}

	if shouldBeAdmin && ((!isSelf && !sender.Admin) || (isSelf && !r.Self.Admin)) {
		return nil, nil, peerNotAdminError(msg.Meta.Sender)
	}

	if isSelf {
		return nil, r.Self, nil
	}

	return sender, nil, nil
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

func peerNotFoundError(peer Fingerprint) error {
	return fmt.Errorf("peer %s not found", string(peer))
}

func peerNotAdminError(peer Fingerprint) error {
	return fmt.Errorf("peer %s is not an admin", string(peer))
}

func ConstructCommand(message []byte, command Command) []byte {
	if command == "" {
		return message
	} else if len(message) == 0 {
		return []byte(command)
	}

	return []byte(string(command) + CommandDelimiter + string(message))
}
