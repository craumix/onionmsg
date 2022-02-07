package types

const startPort = 10048

const (
	defaultSocksPort   = startPort + iota
	defaultControlPort = startPort + iota

	defaultLocalControlPort      = startPort + iota
	defaultLocalConversationPort = startPort + iota

	defaultApiPort = startPort + iota

	numPorts = iota
)

type PortGroup struct {
	TorSocksPort          int
	TorControlPort        int
	LocalControlPort      int
	LocalConversationPort int
	ApiPort               int
}

func NewPortGroup(offset int) PortGroup {
	actualOffset := numPorts * offset

	return PortGroup{
		TorSocksPort:          defaultSocksPort + actualOffset,
		TorControlPort:        defaultControlPort + actualOffset,
		LocalControlPort:      defaultLocalControlPort + actualOffset,
		LocalConversationPort: defaultLocalConversationPort + actualOffset,
		ApiPort:               defaultApiPort + actualOffset,
	}
}
