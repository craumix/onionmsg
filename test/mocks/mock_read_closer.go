package mocks

type MockReadCloser struct {
	ReadFrom        []byte
	ReadReturnError error

	CloseCalled      int
	CloseReturnError error
}

func (m *MockReadCloser) Read(readInto []byte) (int, error) {
	for i, b := range m.ReadFrom {
		readInto[i] = b
	}

	return len(m.ReadFrom), m.ReadReturnError
}

func (m *MockReadCloser) Close() error {
	m.CloseCalled++
	return m.CloseReturnError
}
