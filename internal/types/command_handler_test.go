package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/craumix/onionmsg/internal/types"
)

const testCommand Command = "test-command"

func emptyCommandHandler() CommandHandler {
	return NewCommandHandler(make(map[Command]func(Command, *Message, *Room) error))
}

func TestRegisterCallback(t *testing.T) {
	called := 0
	testFunc := func(Command, *Message, *Room) error {
		called++
		return nil
	}

	testMsg := Message{
		Meta: MessageMeta{},
		Content: MessageContent{
			Type: ContentTypeCmd,
			Blob: &BlobMeta{},
			Data: []byte(testCommand),
		},
		Sig: nil,
	}

	handler := emptyCommandHandler()

	err := handler.RegisterCommand(testCommand, testFunc)
	assert.Nil(t, err)

	err = handler.HandleCommand(&testMsg, nil)
	assert.Nil(t, err)

	assert.Equal(t, 1, called)
}

func TestRegisterCallbackError(t *testing.T) {
	handler := emptyCommandHandler()

	func1 := func() func(Command, *Message, *Room) error {
		return nil
	}

	func2 := func() func(Command, *Message, *Room) error {
		return nil
	}

	err1 := handler.RegisterCommand(testCommand, func1())
	err2 := handler.RegisterCommand(testCommand, func2())

	assert.Nil(t, err1)
	assert.Error(t, err2)
}

func TestHandleCallbackNoCommand(t *testing.T) {
	handler := emptyCommandHandler()

	testMsg := Message{
		Meta: MessageMeta{},
		Content: MessageContent{
			Type: ContentTypeText,
			Blob: &BlobMeta{},
			Data: nil,
		},
		Sig: nil,
	}

	err := handler.HandleCommand(&testMsg, nil)

	assert.Error(t, err)
}

func TestHandleCallbackCommandNotRegistered(t *testing.T) {
	handler := emptyCommandHandler()

	testMsg := Message{
		Meta: MessageMeta{},
		Content: MessageContent{
			Type: ContentTypeCmd,
			Blob: &BlobMeta{},
			Data: []byte(testCommand),
		},
		Sig: nil,
	}

	err := handler.HandleCommand(&testMsg, nil)

	assert.Error(t, err)
}

func TestConstructCommand(t *testing.T) {
	message := "test-message"

	expected := string(testCommand) + CommandDelimiter + message

	actual := ConstructCommand([]byte(message), testCommand)

	assert.Equal(t, expected, string(actual))
}

func TestConstructCommandNoCommand(t *testing.T) {
	message := "test-message"

	actual := ConstructCommand([]byte(message), "")

	assert.Equal(t, message, string(actual))
}

func TestConstructCommandNoMessage(t *testing.T) {
	expected := string(testCommand)

	actual := ConstructCommand([]byte(""), testCommand)

	assert.Equal(t, expected, string(actual))
}
