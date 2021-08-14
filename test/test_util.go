package test

import (
	"encoding/json"
	"errors"
	"github.com/craumix/onionmsg/test/mocks"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func GetRequest(body interface{}, readShouldError, marshalBody bool) *http.Request {
	reader := mocks.MockReadCloser{}

	if readShouldError {
		reader.ReadReturnError = errors.New("test error")
	} else {
		reader.ReadReturnError = io.EOF
	}

	if marshalBody {
		bodyM, _ := json.Marshal(body)
		reader.ReadFrom = bodyM
	} else {
		reader.ReadFrom = []byte(body.(string))
	}

	req, _ := http.NewRequest("", "", &reader)
	req.Form = url.Values{}
	return req
}

func assertZeroStatusCode(t *testing.T, resWriter *mocks.MockResponseWriter, name ...string) {
	assertErrorCode(t, resWriter, 0, name...)
}

func assertErrorCode(t *testing.T, resWriter *mocks.MockResponseWriter, expectedErrorCode int, name ...string) {
	prefix := ""
	for _, s := range name {
		prefix += s
	}
	if len(name) > 0 {
		prefix += ": "
	}

	assert.Equal(t, expectedErrorCode, resWriter.StatusCode, prefix+"Wrong error code was written to header")
}

func assertApplicationJson(t *testing.T, resWriter *mocks.MockResponseWriter) {
	assert.Equal(t, "application/json", resWriter.Head.Get("Content-Type"), "Wrong value in header field Content-Type")
}

func GetTestError() error {
	return errors.New("test error")
}

func GetValidUUID() string {
	return "00000000-0000-0000-0000-000000000000"
}
