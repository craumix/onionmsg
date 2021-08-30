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
	commandCallbacks = map[string]func(Command, *Message, *Room, *RemoteIdentity) error{}
)

func RegisterCommand(command string, callback func(Command, *Message, *Room, *RemoteIdentity) error) error {
	if _, found := commandCallbacks[command]; found {
		return fmt.Errorf("command %s is already registered", command)
	}
	commandCallbacks[command] = callback
	return nil
}

func HandleCommand(message *Message, room *Room, remoteID *RemoteIdentity) error {
	hasCommand, command := message.isCommand()
	if !hasCommand {
		return fmt.Errorf("message doesn't have a command")
	}
	if _, found := commandCallbacks[command]; !found {
		return fmt.Errorf("command %s is not registered", command)
	}
	return commandCallbacks[command](Command(command), message, room, remoteID)
}

func RegisterRoomCommands() error {
	err := RegisterCommand(string(RoomCommandJoin), handleJoin)
	if err != nil {
		return err
	}

	err = RegisterCommand(string(RoomCommandNameRoom), handleNameRoom)
	if err != nil {
		return err
	}

	err = RegisterCommand(string(RoomCommandNick), handleNick)
	if err != nil {
		return err
	}

	return nil
}

func handleJoin(command Command, message *Message, room *Room, _ *RemoteIdentity) error {
	if command != RoomCommandJoin {
		return fmt.Errorf("%s is the wrong command", command)
	}

	args := strings.Split(string(message.Content.Data), CommandDelimiter)
	if !enoughArgs(args, 2) {
		return fmt.Errorf("%s doesn't have enough arguments", RoomCommandJoin)
	}

	if _, ok := room.PeerByFingerprint(args[1]); ok || args[1] == room.Self.Fingerprint() {
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

func handleNameRoom(command Command, message *Message, room *Room, _ *RemoteIdentity) error {
	if command != RoomCommandNameRoom {
		return fmt.Errorf("%s is the wrong command", command)
	}

	args := strings.Split(string(message.Content.Data), CommandDelimiter)
	if !enoughArgs(args, 2) {
		return fmt.Errorf("%s doesn't have enough arguments", RoomCommandNameRoom)
	}

	room.Name = args[1]
	log.Printf("Room with id %s renamed to %s", room.ID, room.Name)

	return nil
}

func handleNick(command Command, message *Message, room *Room, _ *RemoteIdentity) error {
	if command != RoomCommandNick {
		return fmt.Errorf("%s is the wrong command", command)
	}

	args := strings.Split(string(message.Content.Data), CommandDelimiter)
	if !enoughArgs(args, 2) {
		return fmt.Errorf("%s doesn't have enough arguments", RoomCommandNick)
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

func enoughArgs(args []string, needed int) bool {
	if len(args) < needed {
		log.Printf("Not enough args for command \"%s\"\n", strings.Join(args, " "))
		return false
	}
	return true
}
