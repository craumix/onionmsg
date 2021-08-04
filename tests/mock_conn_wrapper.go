package tests

import "github.com/craumix/onionmsg/pkg/sio/connection"

var MockedConn MockConnWrapper
var GetMockedConnWrapperError error

type MockConnWrapper struct {
	WriteBytesInput       []byte
	WriteBytesOutputInt   int
	WriteBytesOutputError error

	ReadBytesOutputBytes []byte
	ReadBytesOutputError error

	WriteStringInput       string
	WriteStringOutputInt   int
	WriteStringOutputError error

	ReadStringOutputString string
	ReadStringOutputError  error

	WriteIntInput       int
	WriteIntOutputInt   int
	WriteIntOutputError error

	ReadIntOutputInt   int
	ReadIntOutputError error

	WriteStructInput       interface{}
	WriteStructOutputInt   int
	WriteStructOutputError error

	ReadStructTargetStruct interface{}
	ReadStructSourceStruct interface{}
	ReadStructOutputError  error

	CloseError  error
	FlushError  error
	BufferedInt int

	Network, Address string
}

func (m MockConnWrapper) WriteBytes(msg []byte) (int, error) {
	m.WriteBytesInput = msg
	return m.WriteBytesOutputInt, m.WriteBytesOutputError
}

func (m MockConnWrapper) ReadBytes() ([]byte, error) {
	return m.ReadBytesOutputBytes, m.ReadBytesOutputError
}

func (m MockConnWrapper) WriteString(msg string) (int, error) {
	m.WriteStringInput = msg
	return m.WriteStringOutputInt, m.WriteStringOutputError
}

func (m MockConnWrapper) ReadString() (string, error) {
	return m.ReadStringOutputString, m.ReadStringOutputError
}

func (m MockConnWrapper) WriteInt(msg int) (int, error) {
	m.WriteIntInput = msg
	return m.WriteIntOutputInt, m.WriteIntOutputError
}

func (m MockConnWrapper) ReadInt() (int, error) {
	return m.ReadIntOutputInt, m.ReadIntOutputError
}

func (m MockConnWrapper) WriteStruct(msg interface{}) (int, error) {
	m.WriteStructInput = msg
	return m.WriteStructOutputInt, m.WriteStructOutputError
}

func (m MockConnWrapper) ReadStruct(target interface{}) error {
	m.ReadStructTargetStruct = target
	target = m.ReadStructSourceStruct
	return m.ReadStructOutputError
}

func (m MockConnWrapper) Flush() error {
	return m.FlushError
}

func (m MockConnWrapper) Close() error {
	return m.CloseError
}

func (m MockConnWrapper) Buffered() int {
	return m.BufferedInt
}

func GetMockedConnWrapper(network, address string) (connection.ConnWrapper, error) {
	return MockedConn, GetMockedConnWrapperError
}
