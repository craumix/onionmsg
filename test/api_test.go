package test

import (
	"encoding/json"
	"fmt"
	"github.com/craumix/onionmsg/internal/api"
	"github.com/craumix/onionmsg/internal/daemon"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/craumix/onionmsg/test/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
)

func TestRouteStatus(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	api.RouteStatus(resWriter, nil)

	assertZeroStatusCode(t, resWriter)
	assertApplicationJson(t, resWriter)
}

func TestRouteTorLog(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	expected := struct {
		Log        string `json:"log"`
		Version    string `json:"version"`
		PID        int    `json:"pid"`
		BinaryPath string `json:"path"`
	}{
		Log:        "Test Log",
		Version:    "1.0",
		PID:        420,
		BinaryPath: "binary/tor",
	}

	daemon.TorInfo = func() interface{} {
		return expected
	}

	api.RouteTorInfo(resWriter, nil)

	actual := struct {
		Log        string `json:"log"`
		Version    string `json:"version"`
		PID        int    `json:"pid"`
		BinaryPath string `json:"path"`
	}{}
	json.Unmarshal(resWriter.WriteInput[0], &actual)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "TorInfo was modified")
}

func TestRouteContactList(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	expected := []string{"Contact1"}

	daemon.ListContactIDs = func() []string {
		return expected
	}

	api.RouteContactList(resWriter, nil)

	var actual []string
	json.Unmarshal(resWriter.WriteInput[0], &actual)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "Contacts were modified!")
}

func TestRouteRoomList(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	expected := []*types.RoomInfo{{
		Self:  "Test Room",
		Peers: nil,
		ID:    uuid.UUID{},
		Name:  "",
		Nicks: nil,
	}}

	daemon.Rooms = func() []*types.RoomInfo {
		return expected
	}

	api.RouteRoomList(resWriter, nil)

	var actual []*types.RoomInfo
	json.Unmarshal(resWriter.WriteInput[0], &actual)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "RoomInfo was modified")
}

func TestRouteRoomCreate(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	var actual []string

	daemon.CreateRoom = func(fingerprints []string) error {
		actual = fingerprints
		return nil
	}

	expected := []string{"id1", "id2"}

	req := GetRequest(expected, false, true)

	api.RouteRoomCreate(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "Fingerprints were modified")
}

func TestRouteRoomCreateErrors(t *testing.T) {
	testcases := []struct {
		name              string
		req               *http.Request
		expectedErrorCode int
	}{
		{
			name:              "ReadAll error",
			req:               GetRequest(nil, true, true),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "Unmarshal error",
			req:               GetRequest("", false, true),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "No ids error",
			req:               GetRequest([]string{}, false, true),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "CreateRoom error",
			req:               GetRequest([]string{"id1"}, false, true),
			expectedErrorCode: http.StatusInternalServerError,
		},
	}

	daemon.CreateRoom = func(fingerprints []string) error {
		return GetTestError()
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		api.RouteRoomCreate(resWriter, tc.req)

		assertErrorCode(t, resWriter, tc.expectedErrorCode, tc.name)
	}
}

func TestDeleteRoom(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	var actual string

	daemon.DeleteRoom = func(uuid string) error {
		actual = uuid
		return nil
	}

	req := GetRequest(nil, false, true)

	expected := "test id"
	req.Form.Add("uuid", expected)

	api.RouteRoomDelete(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "Uuid was modified")
}

func TestDeleteRoomError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.DeleteRoom = func(uuid string) error {
		return GetTestError()
	}

	api.RouteRoomDelete(resWriter, GetRequest(nil, false, true))

	assertErrorCode(t, resWriter, http.StatusInternalServerError)
}

func TestRouteContactCreate(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	expected := "test-id"

	daemon.CreateContactID = func() (string, error) {
		return expected, nil
	}

	api.RouteContactCreate(resWriter, nil)

	assertZeroStatusCode(t, resWriter)
	assertApplicationJson(t, resWriter)
	assert.Equal(t, string(resWriter.WriteInput[0]), fmt.Sprintf("{\"id\":\"%s\"}", expected), "Uuid was modified")
}

func TestRouteContactCreateError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.CreateContactID = func() (string, error) {
		return "", GetTestError()
	}

	api.RouteContactCreate(resWriter, nil)

	assertErrorCode(t, resWriter, http.StatusInternalServerError)
}

func TestRouteContactDelete(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	var actual string

	daemon.DeleteContact = func(fingerprint string) error {
		actual = fingerprint
		return nil
	}

	req := GetRequest(nil, false, true)

	expected := "test id"
	req.Form.Add("id", expected)

	api.RouteContactDelete(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "Uuid was modified")
}

func TestRouteContactDeleteNoID(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	called := false

	daemon.DeleteContact = func(fingerprint string) error {
		called = true
		return nil
	}

	req := GetRequest(nil, false, true)

	api.RouteContactDelete(resWriter, req)

	assertErrorCode(t, resWriter, http.StatusBadRequest)
	assert.False(t, called, "Delete contact got called with missing id field!")
}

func TestRouteContactDeleteError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.DeleteContact = func(fingerprint string) error {
		return GetTestError()
	}

	req := GetRequest(nil, false, true)

	req.Form.Add("id", "test id")

	api.RouteContactDelete(resWriter, req)
	assertErrorCode(t, resWriter, http.StatusInternalServerError)
}

func TestRouteRoomCommandUseradd(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	var actualID, actualFp string

	daemon.AddPeerToRoom = func(roomID uuid.UUID, fingerprint string) error {
		actualID = roomID.String()
		actualFp = fingerprint
		return nil
	}

	expectedFp, expectedID := "test content", GetValidUUID()

	req := GetRequest(expectedFp, false, false)

	req.Form.Add("uuid", expectedID)

	api.RouteRoomCommandUseradd(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expectedID, actualID, "Uuid was modified")
	assert.Equal(t, expectedFp, actualFp, "Fingerprint was modified")
}

func TestRouteRoomCommandUseraddErrors(t *testing.T) {
	testcases := []struct {
		name              string
		req               *http.Request
		uuid              string
		expectedErrorCode int
	}{
		{
			name:              "ReadAll error",
			req:               GetRequest(nil, true, true),
			uuid:              GetValidUUID(),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "Uuid parse error",
			req:               GetRequest(nil, false, true),
			uuid:              "abc",
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "AddPeerToRoom error",
			req:               GetRequest([]string{"test content"}, false, true),
			uuid:              GetValidUUID(),
			expectedErrorCode: http.StatusInternalServerError,
		},
	}

	daemon.AddPeerToRoom = func(roomID uuid.UUID, fingerprint string) error {
		return GetTestError()
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		tc.req.Form.Add("uuid", tc.uuid)

		api.RouteRoomCommandUseradd(resWriter, tc.req)

		assertErrorCode(t, resWriter, tc.expectedErrorCode, tc.name)
	}

}

func TestRoomSendFile(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	blobId := uuid.New()
	blobmngr.MakeBlob = func() (uuid.UUID, error) {
		return blobId, nil
	}

	var calledFileID uuid.UUID
	blobmngr.FileFromID = func(id uuid.UUID) (*os.File, error) {
		calledFileID = id
		return nil, nil
	}

	blobmngr.WriteIntoFile = func(from io.Reader, to *os.File) error {
		return nil
	}

	var (
		calledID      string
		calledType    types.MessageType
		calledContent []byte
	)
	daemon.SendMessage = func(uuid string, msgType types.MessageType, content []byte) error {
		calledID = uuid
		calledType = msgType
		calledContent = content
		return nil
	}

	req := GetRequest(nil, false, true)

	expectedID := "test id"
	req.Form.Add("uuid", expectedID)
	req.Header.Set("Content-Length", "69")

	api.RouteRoomSendFile(resWriter, req)

	// TODO check if the file pointer is the same

	assertZeroStatusCode(t, resWriter)

	assert.Equal(t, calledFileID.String(), blobId.String(), "FileFromID was called with a different id than generated")
	assert.Equal(t, expectedID, calledID)
	assert.Equal(t, types.MessageTypeBlob, calledType)
	assert.Equal(t, string(blobId[:]), string(calledContent))
}

func TestRoomSendFileErrors(t *testing.T) {
	testcases := []struct {
		name             string
		expectedErrCode  int
		fileLength       string
		MakeBlobErr      error
		FileFromIDErr    error
		WriteIntoFileErr error
		SendErr          error
	}{
		{
			name:            "MakeBlobError",
			expectedErrCode: http.StatusInternalServerError,
			MakeBlobErr:     GetTestError(),
		},
		{
			name:            "FileFromIDErr",
			expectedErrCode: http.StatusInternalServerError,
			FileFromIDErr:   GetTestError(),
		},
		{
			name:             "WriteIntoFileErr",
			expectedErrCode:  http.StatusInternalServerError,
			WriteIntoFileErr: GetTestError(),
		},
		{
			name:            "SendErr",
			expectedErrCode: http.StatusBadRequest,
			SendErr:         GetTestError(),
		},
		{
			name:            "FileTooBigErr",
			expectedErrCode: http.StatusBadRequest,
			fileLength:      "2147483700",
		},
		{
			name:            "NotAnIntegerErr",
			expectedErrCode: http.StatusBadRequest,
			fileLength:      "NaN",
		},
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		blobmngr.MakeBlob = func() (uuid.UUID, error) {
			return uuid.UUID{}, tc.MakeBlobErr
		}

		blobmngr.FileFromID = func(id uuid.UUID) (*os.File, error) {
			return nil, tc.FileFromIDErr
		}

		blobmngr.WriteIntoFile = func(from io.Reader, to *os.File) error {
			return tc.WriteIntoFileErr
		}

		daemon.SendMessage = func(uuid string, msgType types.MessageType, content []byte) error {
			return tc.SendErr
		}

		if tc.fileLength == "" {
			tc.fileLength = "42"
		}

		req := GetRequest(nil, false, true)
		req.Header.Set("Content-Length", tc.fileLength)

		api.RouteRoomSendFile(resWriter, req)

		assertErrorCode(t, resWriter, tc.expectedErrCode, tc.name)
	}
}

func TestRouteRoomMessages(t *testing.T) {
	testcases := []struct {
		name            string
		expectedCount   string
		expectedID      string
		ListMessagesErr error
		expectedErrCode int
	}{
		{
			name:          "Count set",
			expectedCount: "42",
		},
		{
			name: "Count not set",
		},
		{
			name:            "Invalid count set",
			expectedCount:   "invalid",
			expectedErrCode: http.StatusBadRequest,
		},
		{
			name:            "ListMessages error",
			ListMessagesErr: GetTestError(),
			expectedErrCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		var actualID string
		daemon.ListMessages = func(uuid string, count int) ([]types.Message, error) {
			actualID = uuid
			return nil, tc.ListMessagesErr
		}

		req := GetRequest(nil, false, true)

		req.Form.Add("count", tc.expectedCount)

		api.RouteRoomMessages(resWriter, req)

		assertErrorCode(t, resWriter, tc.expectedErrCode, tc.name)
		assert.Equal(t, tc.expectedID, actualID, tc.name+": Uuid was modified")
	}
}

func TestRouteBlob(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	var actualID string

	blobmngr.StreamTo = func(id uuid.UUID, w io.Writer) error {
		actualID = id.String()
		return nil
	}

	req := GetRequest(nil, false, true)

	expectedID := GetValidUUID()
	req.Form.Add("uuid", expectedID)

	api.RouteBlob(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expectedID, actualID, "Uuid was modified")
}

func TestRouteBlobErrors(t *testing.T) {
	testcases := []struct {
		name            string
		id              string
		StreamToErr     error
		expectedErrCode int
	}{
		{
			name:            "Invalid uuid",
			id:              "invalid",
			expectedErrCode: http.StatusBadRequest,
		},
		{
			name:            "Stream to error",
			id:              GetValidUUID(),
			expectedErrCode: http.StatusInternalServerError,
			StreamToErr:     GetTestError(),
		},
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		blobmngr.StreamTo = func(id uuid.UUID, w io.Writer) error {
			return tc.StreamToErr
		}

		req := GetRequest(nil, false, true)
		req.Form.Add("uuid", tc.id)

		api.RouteBlob(resWriter, req)

		assertErrorCode(t, resWriter, tc.expectedErrCode, tc.name)
	}
}

func TestSendTextFunctions(t *testing.T) {
	testcases := []struct {
		name            string
		testFunc        func(w http.ResponseWriter, req *http.Request)
		command         types.RoomCommand
		expectedMsgType types.MessageType
	}{
		{
			name:            "RouteRoomCommandSetNick",
			testFunc:        api.RouteRoomCommandSetNick,
			command:         types.RoomCommandNick,
			expectedMsgType: types.MessageTypeCmd,
		},
		{
			name:            "RouteRoomCommandNameRoom",
			testFunc:        api.RouteRoomCommandNameRoom,
			command:         types.RoomCommandNameRoom,
			expectedMsgType: types.MessageTypeCmd,
		},
		{
			name:            "RouteRoomSendMessage",
			testFunc:        api.RouteRoomSendMessage,
			command:         "",
			expectedMsgType: types.MessageTypeText,
		},
	}

	var (
		actualID      string
		actualMsgType types.MessageType
		actualContent []byte
	)

	daemon.SendMessage = func(uuid string, msgType types.MessageType, content []byte) error {
		actualID = uuid
		actualMsgType = msgType
		actualContent = content
		return nil
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		reader := mocks.MockReadCloser{}
		reader.ReadReturnError = io.EOF

		expectedContent := "test content"
		reader.ReadFrom = []byte(expectedContent)
		if tc.command != "" {
			expectedContent = string(tc.command) + " " + expectedContent
		}

		req, _ := http.NewRequest("", "", &reader)

		expectedID := "test id"
		req.Form = url.Values{}
		req.Form.Add("uuid", expectedID)

		tc.testFunc(resWriter, req)

		assertZeroStatusCode(t, resWriter)

		assert.Equal(t, expectedID, actualID, tc.name+": Uuid was modified")
		assert.Equal(t, tc.expectedMsgType, actualMsgType, tc.name+": Wrong message type")
		assert.Equal(t, expectedContent, string(actualContent), tc.name+": Wrong content")
	}
}

func TestSendTextFunctionsErrors(t *testing.T) {
	testcases := []struct {
		name     string
		testFunc func(w http.ResponseWriter, req *http.Request)
	}{
		{
			name:     "RouteRoomCommandSetNick",
			testFunc: api.RouteRoomCommandSetNick,
		},
		{
			name:     "RouteRoomCommandNameRoom",
			testFunc: api.RouteRoomCommandNameRoom,
		},
		{
			name:     "RouteRoomSendMessage",
			testFunc: api.RouteRoomSendMessage,
		},
	}

	testErrors := []struct {
		name              string
		req               *http.Request
		expectedErrorCode int
	}{
		{
			name:              "ReadAllError",
			req:               GetRequest(nil, true, true),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "SendError",
			req:               GetRequest([]string{"test content"}, false, true),
			expectedErrorCode: http.StatusInternalServerError,
		},
	}

	daemon.SendMessage = func(uuid string, msgType types.MessageType, content []byte) error {
		return GetTestError()
	}

	for _, tc := range testcases {
		for _, te := range testErrors {
			resWriter := mocks.GetMockResponseWriter()

			tc.testFunc(resWriter, te.req)

			assertErrorCode(t, resWriter, te.expectedErrorCode, tc.name+"-"+te.name)
		}
	}
}
