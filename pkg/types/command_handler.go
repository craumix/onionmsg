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

	return nil
}

func joinCallback(command Command, message *Message, room *Room, _ *RemoteIdentity) error {
	args, err := parseCommand(message, command, RoomCommandJoin, 2)
	if err != nil {
		return err
	}

	if _, ok := room.PeerByFingerprint(args[1]); ok || args[1] == Fingerprint(room.Self) {
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
	if found {
		nickname := args[1]
		identity.Nick = nickname
		log.Printf("Set nickname for %s to %s", sender, nickname)
	} else {
		return fmt.Errorf("peer %s not found", sender)
	}

	return nil
}

func parseCommand(message *Message, actualCommand, expectedCommand Command, neededArgs int) ([]string, error) {
	if actualCommand != expectedCommand {
		return nil, fmt.Errorf("%s is the wrong command", actualCommand)
	}

	args := strings.Split(string(message.Content.Data), CommandDelimiter)
	if !enoughArgs(args, neededArgs) {
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

func AddCommand(message []byte, command Command) []byte {
	return []byte(string(command) + CommandDelimiter + string(message))
}

func CleanCallbacks() {
	commandCallbacks = map[Command]func(Command, *Message, *Room, *RemoteIdentity) error{}
}
