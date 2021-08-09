package mocks

import (
	"net/http"
)

type MockResponseWriter struct {
	Head http.Header

	HeaderCalled bool

	WriteInput       [][]byte
	WriteOutputInt   int
	WriteOutputError error

	WriteHeaderCalled bool
	StatusCode        int
}

func (m *MockResponseWriter) Header() http.Header {
	m.HeaderCalled = true
	return m.Head
}

func (m *MockResponseWriter) Write(bytes []byte) (int, error) {
	m.WriteInput = append(m.WriteInput, bytes)
	return m.WriteOutputInt, m.WriteOutputError
}

func (m *MockResponseWriter) WriteHeader(statusCode int) {
	m.WriteHeaderCalled = true
	m.StatusCode = statusCode
}

func GetMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		Head: http.Header{},
	}
}
