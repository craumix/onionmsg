package connection

type ConnWrapper interface {
	WriteBytes(msg []byte) (int, error)
	ReadBytes() ([]byte, error)

	WriteString(msg string) (int, error)
	ReadString() (string, error)

	WriteInt(msg int) (int, error)
	ReadInt() (int, error)

	WriteStruct(msg interface{}) (int, error)
	ReadStruct(target interface{}) error

	Flush() error
	Close() error
	Buffered() int
}
