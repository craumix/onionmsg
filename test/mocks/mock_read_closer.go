package mocks

type MockReadCloser struct {
	ReadFrom        []byte
	ReadReturnError error

	CloseCalled      int
	CloseReturnError error
}

func (m *MockReadCloser) Read(readInto []byte) (n int, err error) {
	for i, byte := range m.ReadFrom {
		readInto[i] = byte
	}

	return len(m.ReadFrom), m.ReadReturnError
}

func (m *MockReadCloser) Close() error {
	m.CloseCalled++
	return m.CloseReturnError
}
