package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/craumix/onionmsg/internal/api"
	"github.com/craumix/onionmsg/internal/daemon"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/pkg/types"
	"github.com/craumix/onionmsg/test/mocks"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
)

func TestRouteStatus(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	api.RouteStatus(resWriter, nil)

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	contentType := resWriter.Head.Get("Content-Type")
	if contentType != "application/json" {
		if contentType == "" {
			t.Error("Content-Type not set in header!")
		} else {
			t.Errorf("Wrong value of Content-Type %s instead of application/json", contentType)
		}
	}
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

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	if actual.Log != expected.Log {
		t.Error("Incorrect Log!")
	}

	if actual.Version != expected.Version {
		t.Error("Incorrect Version!")
	}

	if actual.PID != expected.PID {
		t.Error("Incorrect PID!")
	}

	if actual.BinaryPath != expected.BinaryPath {
		t.Error("Incorrect Binary Path!")
	}
}

func TestRouteContactList(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.ListContactIDs = func() []string {
		return []string{"Contact1"}
	}

	api.RouteContactList(resWriter, nil)

	var written []string

	json.Unmarshal(resWriter.WriteInput[0], &written)

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	if written[0] != "Contact1" {
		t.Errorf("Wrong contacts!")
	}
}

func TestRouteRoomList(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.Rooms = func() []*types.RoomInfo {
		return []*types.RoomInfo{
			&types.RoomInfo{
				Self:  "Test Room",
				Peers: nil,
				ID:    uuid.UUID{},
				Name:  "",
				Nicks: nil,
			},
		}
	}

	api.RouteRoomList(resWriter, nil)

	var written []*types.RoomInfo

	json.Unmarshal(resWriter.WriteInput[0], &written)

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	if written[0].Self != "Test Room" {
		t.Errorf("Wrong RoomInfo!")
	}
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

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	if !SameStringArray(expected, actual) {
		t.Errorf("Got %v instead of %v!", actual, expected)
	}
}

func TestRouteRoomCreateErrors(t *testing.T) {
	testCases := []struct {
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
		return errors.New("test error")
	}

	for _, testCase := range testCases {
		resWriter := mocks.GetMockResponseWriter()

		api.RouteRoomCreate(resWriter, testCase.req)

		if testCase.expectedErrorCode != resWriter.StatusCode {
			t.Errorf("%s got %d instead of %d!", testCase.name, resWriter.StatusCode, testCase.expectedErrorCode)
		}
	}
}

func TestDeleteRoom(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	actual := ""
	daemon.DeleteRoom = func(uuid string) error {
		actual = uuid
		return nil
	}

	req := GetRequest(nil, false, true)

	expected := "test id"
	req.Form.Add("uuid", expected)

	api.RouteRoomDelete(resWriter, req)

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	if actual != expected {
		t.Errorf("Got wrong uuid %s!", actual)
	}
}

func TestDeleteRoomError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.DeleteRoom = func(uuid string) error {
		return errors.New("test error")
	}

	api.RouteRoomDelete(resWriter, GetRequest(nil, false, true))

	if resWriter.StatusCode != http.StatusInternalServerError {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}
}

func TestRouteContactCreate(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	expected := "test-id"
	daemon.CreateContactID = func() (string, error) {
		return expected, nil
	}

	api.RouteContactCreate(resWriter, nil)

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	contentType := resWriter.Head.Get("Content-Type")
	if contentType != "application/json" {
		if contentType == "" {
			t.Error("Content-Type not set in header!")
		} else {
			t.Errorf("Wrong value of Content-Type %s instead of application/json", contentType)
		}
	}

	if string(resWriter.WriteInput[0]) != fmt.Sprintf("{\"id\":\"%s\"}", expected) {
		t.Error("Id is not being written properly")
	}
}

func TestRouteContactCreateError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.CreateContactID = func() (string, error) {
		return "", errors.New("test error")
	}

	api.RouteContactCreate(resWriter, nil)

	if resWriter.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong error code got %d instead of %d!", resWriter.StatusCode, http.StatusInternalServerError)
	}
}

func TestRouteContactDelete(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	actual := ""
	daemon.DeleteContact = func(fingerprint string) error {
		actual = fingerprint
		return nil
	}

	req := GetRequest(nil, false, true)

	expected := "test id"
	req.Form.Add("id", expected)

	api.RouteContactDelete(resWriter, req)

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	if actual != expected {
		t.Errorf("Got wrong id %s!", actual)
	}
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

	if resWriter.StatusCode != http.StatusBadRequest {
		t.Errorf("Wrong error code got %d instead of %d!", resWriter.StatusCode, http.StatusBadRequest)
	}

	if called {
		t.Error("Delete contact got called with missing id field!")
	}
}

func TestRouteContactDeleteError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.DeleteContact = func(fingerprint string) error {
		return errors.New("test error")
	}

	req := GetRequest(nil, false, true)

	req.Form.Add("id", "test id")

	api.RouteContactDelete(resWriter, req)

	if resWriter.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong error code got %d instead of %d!", resWriter.StatusCode, http.StatusInternalServerError)
	}
}

func TestRouteRoomCommandUseradd(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	var (
		calledID = ""
		calledFp = ""
	)

	daemon.AddPeerToRoom = func(roomID uuid.UUID, fingerprint string) error {
		calledID = roomID.String()
		calledFp = fingerprint
		return nil
	}

	expectedFp := "test content"
	req := GetRequest(expectedFp, false, false)

	expectedID := "00000000-0000-0000-0000-000000000000"
	req.Form.Add("uuid", expectedID)

	api.RouteRoomCommandUseradd(resWriter, req)

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	if calledID != expectedID {
		t.Errorf("%s is not %s", calledID, expectedID)
	}

	if calledFp != expectedFp {
		t.Errorf("%s is not %s", calledFp, expectedFp)
	}

}

func TestRouteRoomCommandUseraddErrors(t *testing.T) {
	testCases := []struct {
		name              string
		req               *http.Request
		uuid              string
		expectedErrorCode int
	}{
		{
			name:              "ReadAll error",
			req:               GetRequest(nil, true, true),
			uuid:              "00000000-0000-0000-0000-000000000000",
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
			uuid:              "00000000-0000-0000-0000-000000000000",
			expectedErrorCode: http.StatusInternalServerError,
		},
	}

	daemon.AddPeerToRoom = func(roomID uuid.UUID, fingerprint string) error {
		return errors.New("test error")
	}

	for _, testCase := range testCases {
		resWriter := mocks.GetMockResponseWriter()

		testCase.req.Form.Add("uuid", testCase.uuid)

		api.RouteRoomCommandUseradd(resWriter, testCase.req)

		if testCase.expectedErrorCode != resWriter.StatusCode {
			t.Errorf("%s got %d instead of %d!", testCase.name, resWriter.StatusCode, testCase.expectedErrorCode)
		}
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

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	if blobId != calledFileID {
		t.Errorf("FileFromID was called with a different id than generated")
	}

	if calledID != expectedID {
		t.Errorf("Got wrong uuid %s!", calledID)
	}

	if calledType != types.MessageTypeBlob {
		t.Errorf("Got wrong Message txpe got %s instead of %s", calledType, types.MessageTypeBlob)
	}

	if !SameByteArray(calledContent, blobId[:]) {
		t.Errorf("Got wrong content got %s instead of %s", calledContent, blobId.String())
	}

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
			MakeBlobErr:     errors.New("test error"),
		},
		{
			name:            "FileFromIDErr",
			expectedErrCode: http.StatusInternalServerError,
			FileFromIDErr:   errors.New("test error"),
		},
		{
			name:             "WriteIntoFileErr",
			expectedErrCode:  http.StatusInternalServerError,
			WriteIntoFileErr: errors.New("test error"),
		},
		{
			name:            "SendErr",
			expectedErrCode: http.StatusBadRequest,
			SendErr:         errors.New("test error"),
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

		if resWriter.StatusCode != tc.expectedErrCode {
			t.Errorf("%s got %d instead of %d", tc.name, resWriter.StatusCode, tc.expectedErrCode)
		}

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
			ListMessagesErr: errors.New("test error"),
			expectedErrCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		var (
			calledID    string
			calledCount int
		)
		daemon.ListMessages = func(uuid string, count int) ([]types.Message, error) {
			calledID = uuid
			calledCount = count
			return nil, tc.ListMessagesErr
		}

		req := GetRequest(nil, false, true)

		req.Form.Add("count", tc.expectedCount)

		api.RouteRoomMessages(resWriter, req)

		if resWriter.StatusCode != tc.expectedErrCode {
			t.Errorf("%s got unexpected error %d!", tc.name, resWriter.StatusCode)
		}

		if calledID != tc.expectedID {
			t.Errorf("%s got wrong uuid %s!", tc.name, calledID)
		}

		calledCountInt, _ := strconv.Atoi(tc.expectedCount)
		if calledCount != calledCountInt {
			t.Errorf("%s got wrong count %d!", tc.name, calledCount)
		}
	}
}

func TestRouteBlob(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	var (
		calledID string
	)

	blobmngr.StreamTo = func(id uuid.UUID, w io.Writer) error {
		calledID = id.String()
		return nil
	}

	req := GetRequest(nil, false, true)

	expectedID := "00000000-0000-0000-0000-000000000000"
	req.Form.Add("uuid", expectedID)

	api.RouteBlob(resWriter, req)

	if resWriter.StatusCode != 0 {
		t.Errorf("Got unexpected error %d!", resWriter.StatusCode)
	}

	if calledID != expectedID {
		t.Errorf("%s is not %s", calledID, expectedID)
	}
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
			id:              "00000000-0000-0000-0000-000000000000",
			expectedErrCode: http.StatusInternalServerError,
			StreamToErr:     errors.New("test error"),
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

		if resWriter.StatusCode != tc.expectedErrCode {
			t.Errorf("%s got %d instead of %d", tc.name, resWriter.StatusCode, tc.expectedErrCode)
		}
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

		if resWriter.StatusCode != 0 {
			t.Errorf("%s got unexpected error %d!", tc.name, resWriter.StatusCode)
		}

		if calledID != expectedID {
			t.Errorf("%s got wrong uuid %s!", tc.name, calledID)
		}

		if calledType != tc.expectedMsgType {
			t.Errorf("%s got wrong Message txpe got %s instead of %s", tc.name, calledType, tc.expectedMsgType)
		}

		if !SameByteArray(calledContent, []byte(expectedContent)) {
			t.Errorf("%s got wrong content got %s instead of %s", tc.name, calledContent, expectedContent)
		}
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
		return errors.New("test error")
	}

	for _, tc := range testcases {
		for _, te := range testErrors {
			resWriter := mocks.GetMockResponseWriter()

			tc.testFunc(resWriter, te.req)

			if te.expectedErrorCode != resWriter.StatusCode {
				t.Errorf("%s %s got %d instead of %d!",
					tc.name, te.name,
					resWriter.StatusCode,
					te.expectedErrorCode,
				)
			}
		}
	}
}