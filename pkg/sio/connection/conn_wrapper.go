package connection

type ConnWrapper interface {
	WriteBytes(msg []byte, compress bool) (int, error)
	ReadBytes(compressed bool) ([]byte, error)

	WriteString(msg string) (int, error)
	ReadString() (string, error)

	WriteInt(msg int) (int, error)
	ReadInt() (int, error)

	WriteStruct(msg interface{}, compress bool) (int, error)
	ReadStruct(target interface{}, compressed bool) error

	Flush() error
	Close() error
	Buffered() int
}
