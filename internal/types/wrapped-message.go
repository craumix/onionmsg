package types

type WrappedMessage struct {
	Message	*Message
	Remote	*RemoteIdentity
}

func NewWrappedMessage(msg *Message, remote *RemoteIdentity) *WrappedMessage {
	return &WrappedMessage{
		Message: msg,
		Remote: remote,
	}
}

func BulkNewWrappedMessage(msg *Message, remotes []*RemoteIdentity) []*WrappedMessage {
	messages := make([]*WrappedMessage, len(remotes))
	for i, r := range remotes {
		messages[i] = NewWrappedMessage(msg, r)
	}
	return messages
}