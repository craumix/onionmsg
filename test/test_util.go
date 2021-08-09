package test

import (
	"encoding/json"
	"errors"
	"github.com/craumix/onionmsg/test/mocks"
	"io"
	"net/http"
	"net/url"
)

func SameByteArray(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		// println("%b\n%b", a[i], b[i])
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func SameStringArray(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		// println("%b\n%b", a[i], b[i])
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

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
