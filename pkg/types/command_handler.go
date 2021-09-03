package types

import (
	"fmt"
	"log"
	"strings"
)

type Command string

const (
	RoomCommandJoin     Command = "join"
	RoomCommandNameRoom Command = "name_room"
	RoomCommandNick     Command = "nick"
	RoomCommandPromote  Command = "promote"

	CommandDelimiter = " "
)

var (
	commandCallbacks = map[Command]func(Command, *Message, *Room, *RemoteIdentity) error{}
)

func RegisterCommand(command Command, callback func(Command, *Message, *Room, *RemoteIdentity) error) error {
	if _, found := commandCallbacks[command]; found {
		return fmt.Errorf("command %s is already registered", command)
	}
	commandCallbacks[command] = callback
	return nil
}

func HandleCommand(message *Message, room *Room, remoteID *RemoteIdentity) error {
	hasCommand, command := message.isCommand()
	if !hasCommand {
		return fmt.Errorf("message isn't a command")
	}
	if _, found := commandCallbacks[Command(command)]; !found {
		return fmt.Errorf("command %s is not registered", command)
	}
	return commandCallbacks[Command(command)](Command(command), message, room, remoteID)
}

func RegisterRoomCommands() error {
	err := RegisterCommand(RoomCommandJoin, joinCallback)
	if err != nil {
		return err
	}

	err = RegisterCommand(RoomCommandNameRoom, nameRoomCallback)
	if err != nil {
		return err
	}

	err = RegisterCommand(RoomCommandNick, nickCallback)
	if err != nil {
		return err
	}

	err = RegisterCommand(RoomCommandPromote, promoteCallback)
	if err != nil {
		return err
	}

	return nil
}

func joinCallback(command Command, message *Message, room *Room, _ *RemoteIdentity) error {
	args, err := parseCommand(message, command, RoomCommandJoin, 2)
	if err != nil {
		return err
	}

	if _, found := room.PeerByFingerprint(args[1]); found || args[1] == room.Self.Fingerprint() {
		return fmt.Errorf("user %s already added, or self", args[1])
	}

	peerID, err := NewRemoteIdentity(args[1])
	if err != nil {
		return err
	}

	newPeer := NewMessagingPeer(peerID)
	room.Peers = append(room.Peers, newPeer)

	go newPeer.RunMessageQueue(room.Ctx, room)

	log.Printf("New peer %s added to Room %s\n", newPeer.RIdentity.Fingerprint(), room.ID)
	return nil
}

func nameRoomCallback(command Command, message *Message, room *Room, _ *RemoteIdentity) error {
	args, err := parseCommand(message, command, RoomCommandNameRoom, 2)
	if err != nil {
		return err
	}

	room.Name = args[1]
	log.Printf("Room with id %s renamed to %s", room.ID, room.Name)

	return nil
}

func nickCallback(command Command, message *Message, room *Room, _ *RemoteIdentity) error {
	args, err := parseCommand(message, command, RoomCommandNick, 2)
	if err != nil {
		return err
	}

	sender := message.Meta.Sender
	identity, found := room.PeerByFingerprint(sender)
	if !found {
		return peerNotFoundError(sender)
	}

	nickname := args[1]
	identity.Nick = nickname
	log.Printf("Set nickname for %s to %s", sender, nickname)

	return nil
}

func promoteCallback(command Command, message *Message, room *Room, _ *RemoteIdentity) error {
	args, err := parseCommand(message, command, RoomCommandPromote, 2)
	if err != nil {
		return err
	}

	sender, found := room.PeerByFingerprint(message.Meta.Sender)
	if !found {
		return peerNotFoundError(message.Meta.Sender)
	} else if !sender.Admin {
		return peerNotAdminError(message.Meta.Sender)
	}

	toPromote, found := room.PeerByFingerprint(args[1])
	switch {
	case found:
		toPromote.Admin = true
	case room.isSelf(args[1]):
		room.Self.Admin = true
	default:
		return peerNotFoundError(args[1])
	}

	return nil
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
		log.Printf("Not enough args for command \"%s\"\n", strings.Join(args, " "))
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

func AddCommand(message []byte, command Command) []byte {
	return []byte(string(command) + CommandDelimiter + string(message))
}

func CleanCallbacks() {
	commandCallbacks = map[Command]func(Command, *Message, *Room, *RemoteIdentity) error{}
}
