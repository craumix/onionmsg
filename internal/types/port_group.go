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
	SocksPort             int
	ControlPort           int
	LocalControlPort      int
	LocalConversationPort int
	ApiPort               int
}

func NewPortGroup(offset int) PortGroup {
	actualOffset := numPorts * offset

	return PortGroup{
		SocksPort:             defaultSocksPort + actualOffset,
		ControlPort:           defaultControlPort + actualOffset,
		LocalControlPort:      defaultLocalControlPort + actualOffset,
		LocalConversationPort: defaultLocalConversationPort + actualOffset,
		ApiPort:               defaultApiPort + actualOffset,
	}
}
