package types_test

import (
	"github.com/stretchr/testify/assert"
	"testing"
)
import . "github.com/craumix/onionmsg/pkg/types"

const testCommand Command = "test-command"

func TestRegisterCallback(t *testing.T) {
	CleanCallbacks()

	called := 0
	RegisterCommand(testCommand, func(Command, *Message, *Room) error {
		called++
		return nil
	})

	testMsg := Message{
		Meta: MessageMeta{},
		Content: MessageContent{
			Type: ContentTypeCmd,
			Blob: &BlobMeta{},
			Data: []byte(testCommand),
		},
		Sig: nil,
	}

	HandleCommand(&testMsg, nil)

	assert.Equal(t, 1, called)
}

func TestRegisterCallbackError(t *testing.T) {
	CleanCallbacks()

	err1 := RegisterCommand(testCommand, func(Command, *Message, *Room) error {
		return nil
	})

	err2 := RegisterCommand(testCommand, func(Command, *Message, *Room) error {
		return nil
	})

	assert.Nil(t, err1)
	assert.Error(t, err2)
}

func TestHandleCallbackNoCommand(t *testing.T) {
	CleanCallbacks()

	testMsg := Message{
		Meta: MessageMeta{},
		Content: MessageContent{
			Type: ContentTypeText,
			Blob: &BlobMeta{},
			Data: nil,
		},
		Sig: nil,
	}

	actual := HandleCommand(&testMsg, nil)

	assert.Error(t, actual)
}

func TestHandleCallbackCommandNotRegistered(t *testing.T) {
	CleanCallbacks()

	testMsg := Message{
		Meta: MessageMeta{},
		Content: MessageContent{
			Type: ContentTypeCmd,
			Blob: &BlobMeta{},
			Data: []byte(testCommand),
		},
		Sig: nil,
	}

	actual := HandleCommand(&testMsg, nil)

	assert.Error(t, actual)
}

func TestCleanCallbacks(t *testing.T) {
	CleanCallbacks()

	called := 0
	RegisterCommand(testCommand, func(Command, *Message, *Room) error {
		called++
		return nil
	})

	testMsg := Message{
		Meta: MessageMeta{},
		Content: MessageContent{
			Type: ContentTypeCmd,
			Blob: &BlobMeta{},
			Data: []byte(testCommand),
		},
		Sig: nil,
	}

	CleanCallbacks()

	actual := HandleCommand(&testMsg, nil)

	assert.Error(t, actual)
	assert.Zero(t, called)
}

func TestAddCommand(t *testing.T) {
	message := "test-message"

	expected := string(testCommand) + CommandDelimiter + message

	actual := ConstructCommand([]byte(message), testCommand)

	assert.Equal(t, expected, string(actual))
}
