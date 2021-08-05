package tests

import (
	"encoding/json"
	"github.com/craumix/onionmsg/pkg/sio/connection"
)

var MockedConn *MockConnWrapper

type MockConnWrapper struct {
	WriteBytesInput       [][]byte
	WriteBytesOutputInt   int
	WriteBytesOutputError error

	ReadBytesOutputBytes []byte
	ReadBytesOutputError error

	WriteStringInput       []string
	WriteStringOutputInt   int
	WriteStringOutputError error

	ReadStringOutputString string
	ReadStringOutputError  error

	WriteIntInput       []int
	WriteIntOutputInt   int
	WriteIntOutputError error

	ReadIntOutputInt   int
	ReadIntOutputError error

	WriteStructInput       []interface{}
	WriteStructOutputInt   int
	WriteStructOutputError error

	ReadStructTargetStruct []interface{}
	ReadStructSourceStruct interface{}
	ReadStructOutputError  error

	CloseError  error
	CloseCalled bool

	FlushError  error
	FlushCalled bool

	BufferedInt    int
	BufferedCalled bool

	Network, Address           string
	GetMockedConnWrapperCalled bool
	GetMockedConnWrapperError  error
}

func (m *MockConnWrapper) WriteBytes(msg []byte) (int, error) {
	m.WriteBytesInput = append(m.WriteBytesInput, msg)

	return m.WriteBytesOutputInt, m.WriteBytesOutputError
}

func (m *MockConnWrapper) ReadBytes() ([]byte, error) {
	return m.ReadBytesOutputBytes, m.ReadBytesOutputError
}

func (m *MockConnWrapper) WriteString(msg string) (int, error) {
	m.WriteStringInput = append(m.WriteStringInput, msg)
	return m.WriteStringOutputInt, m.WriteStringOutputError
}

func (m *MockConnWrapper) ReadString() (string, error) {
	return m.ReadStringOutputString, m.ReadStringOutputError
}

func (m *MockConnWrapper) WriteInt(msg int) (int, error) {
	m.WriteIntInput = append(m.WriteIntInput, msg)
	return m.WriteIntOutputInt, m.WriteIntOutputError
}

func (m *MockConnWrapper) ReadInt() (int, error) {
	return m.ReadIntOutputInt, m.ReadIntOutputError
}

func (m *MockConnWrapper) WriteStruct(msg interface{}) (int, error) {
	m.WriteStructInput = append(m.WriteStructInput, msg)
	return m.WriteStructOutputInt, m.WriteStructOutputError
}

func (m *MockConnWrapper) ReadStruct(target interface{}) error {
	m.ReadStructTargetStruct = append(m.ReadStructTargetStruct, target)
	raw, _ := json.Marshal(m.ReadStructSourceStruct)
	json.Unmarshal(raw, target)
	return m.ReadStructOutputError
}

func (m *MockConnWrapper) Flush() error {
	m.FlushCalled = true
	return m.FlushError
}

func (m *MockConnWrapper) Close() error {
	m.CloseCalled = true
	return m.CloseError
}

func (m *MockConnWrapper) Buffered() int {
	m.BufferedCalled = true
	return m.BufferedInt
}

func GetMockedConnWrapper(network, address string) (connection.ConnWrapper, error) {
	MockedConn.GetMockedConnWrapperCalled = true
	return MockedConn, MockedConn.GetMockedConnWrapperError
}
