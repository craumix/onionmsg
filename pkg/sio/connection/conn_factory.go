package connection

var (
	GetConnFunc func(network, address string) (ConnWrapper, error)
)
