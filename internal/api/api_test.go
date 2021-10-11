package api_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/craumix/onionmsg/internal/api"
	"github.com/craumix/onionmsg/internal/daemon"
	"github.com/craumix/onionmsg/internal/types"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/craumix/onionmsg/test"
	"github.com/craumix/onionmsg/test/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

	req := getRequest(expected, false, true)

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
			req:               getRequest(nil, true, true),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "Unmarshal error",
			req:               getRequest("", false, true),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "No ids error",
			req:               getRequest([]string{}, false, true),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "CreateRoom error",
			req:               getRequest([]string{"id1"}, false, true),
			expectedErrorCode: http.StatusInternalServerError,
		},
	}

	daemon.CreateRoom = func(fingerprints []string) error {
		return test.GetTestError()
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

	req := getRequest(nil, false, true)

	expected := "test id"
	req.Form.Add("uuid", expected)

	api.RouteRoomDelete(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "Uuid was modified")
}

func TestDeleteRoomError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.DeleteRoom = func(uuid string) error {
		return test.GetTestError()
	}

	api.RouteRoomDelete(resWriter, getRequest(nil, false, true))

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
		return "", test.GetTestError()
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

	req := getRequest(nil, false, true)

	expected := "test id"
	req.Form.Add("fingerprint", expected)

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

	req := getRequest(nil, false, true)

	api.RouteContactDelete(resWriter, req)

	assertErrorCode(t, resWriter, http.StatusBadRequest)
	assert.False(t, called, "Delete contact got called with missing id field!")
}

func TestRouteContactDeleteError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	daemon.DeleteContact = func(fingerprint string) error {
		return test.GetTestError()
	}

	req := getRequest(nil, false, true)

	req.Form.Add("fingerprint", "test id")

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

	expectedFp, expectedID := "test content", test.GetValidUUID()

	req := getRequest(expectedFp, false, false)

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
			req:               getRequest(nil, true, true),
			uuid:              test.GetValidUUID(),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "Uuid parse error",
			req:               getRequest(nil, false, true),
			uuid:              "abc",
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "AddPeerToRoom error",
			req:               getRequest([]string{"test content"}, false, true),
			uuid:              test.GetValidUUID(),
			expectedErrorCode: http.StatusInternalServerError,
		},
	}

	daemon.AddPeerToRoom = func(roomID uuid.UUID, fingerprint string) error {
		return test.GetTestError()
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

	newBlobId := uuid.New()
	blobmngr.MakeBlob = func() (uuid.UUID, error) {
		return newBlobId, nil
	}

	var actualFileId uuid.UUID
	blobmngr.FileFromID = func(id uuid.UUID) (*os.File, error) {
		actualFileId = id
		return nil, nil
	}

	blobmngr.WriteIntoFile = func(from io.Reader, to *os.File) error {
		return nil
	}

	var (
		actualID         string
		actualMsgContent types.MessageContent
	)
	daemon.SendMessage = func(uuid string, msgContent types.MessageContent) error {
		actualID = uuid
		actualMsgContent = msgContent
		return nil
	}

	req := getRequest(nil, false, true)

	expectedID := "test id"
	req.Form.Add("uuid", expectedID)

	expectedMsgContent := types.MessageContent{
		Type: types.ContentTypeFile,
		Blob: &types.BlobMeta{
			ID:   newBlobId,
			Name: "test-filename",
			Type: "test-mimetype",
		},
		Data: nil,
	}

	req.Header.Set(api.FilenameHeader, expectedMsgContent.Blob.Name)
	req.Header.Set(api.MimetypeHeader, expectedMsgContent.Blob.Type)
	req.Header.Set("Content-Length", "69")

	api.RouteRoomSendFile(resWriter, req)

	// TODO check if the file pointer is the same

	assertZeroStatusCode(t, resWriter)

	assert.Equal(t, newBlobId.String(), actualFileId.String(), "FileFromID was called with a different id than generated")
	assert.Equal(t, expectedID, actualID, "SendMessage didn't get the Id from the request")
	assert.Equal(t, expectedMsgContent, actualMsgContent)
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
			MakeBlobErr:     test.GetTestError(),
		},
		{
			name:            "FileFromIDErr",
			expectedErrCode: http.StatusInternalServerError,
			FileFromIDErr:   test.GetTestError(),
		},
		{
			name:             "WriteIntoFileErr",
			expectedErrCode:  http.StatusInternalServerError,
			WriteIntoFileErr: test.GetTestError(),
		},
		{
			name:            "SendErr",
			expectedErrCode: http.StatusBadRequest,
			SendErr:         test.GetTestError(),
		},
		/*
			Filesize is not checked atm
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
		*/
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

		daemon.SendMessage = func(uuid string, content types.MessageContent) error {
			return tc.SendErr
		}

		if tc.fileLength == "" {
			tc.fileLength = "42"
		}

		req := getRequest(nil, false, true)
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
			ListMessagesErr: test.GetTestError(),
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

		req := getRequest(nil, false, true)

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

	blobmngr.StatFromID = func(id uuid.UUID) (fs.FileInfo, error) {
		return nil, nil
	}

	req := getRequest(nil, false, true)

	expectedID := test.GetValidUUID()
	req.Form.Add("uuid", expectedID)

	api.RouteBlob(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expectedID, actualID, "Uuid was modified")
	assert.Equal(t, "public, max-age=604800, immutable", resWriter.Head.Get("Cache-Control"))
}

func TestRouteBlobStreamToErrors(t *testing.T) {
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
			id:              test.GetValidUUID(),
			expectedErrCode: http.StatusInternalServerError,
			StreamToErr:     test.GetTestError(),
		},
	}

	blobmngr.StatFromID = func(id uuid.UUID) (fs.FileInfo, error) {
		return nil, nil
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		blobmngr.StreamTo = func(id uuid.UUID, w io.Writer) error {
			return tc.StreamToErr
		}

		req := getRequest(nil, false, true)
		req.Form.Add("uuid", tc.id)

		api.RouteBlob(resWriter, req)

		assertErrorCode(t, resWriter, tc.expectedErrCode, tc.name)
	}
}

func TestRouteBlobBlobNotFoundError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	called := false
	blobmngr.StreamTo = func(id uuid.UUID, w io.Writer) error {
		called = true
		return nil
	}

	blobmngr.StatFromID = func(id uuid.UUID) (fs.FileInfo, error) {
		return nil, os.ErrNotExist
	}

	req := getRequest(nil, false, true)

	expectedID := test.GetValidUUID()
	req.Form.Add("uuid", expectedID)

	api.RouteBlob(resWriter, req)

	assertErrorCode(t, resWriter, http.StatusNotFound)
	assert.False(t, called)
}

func TestRouteBlobContentDisposition(t *testing.T) {

	testcases := []struct {
		name                       string
		filename                   string
		expectedContentDisposition string
	}{
		{
			name:                       "Empty filename header",
			filename:                   "",
			expectedContentDisposition: "",
		},
		{
			name:                       "Set filename header",
			filename:                   "test",
			expectedContentDisposition: "attachment; filename=\"test\"",
		},
	}

	blobmngr.StreamTo = func(id uuid.UUID, w io.Writer) error {
		return nil
	}

	blobmngr.StatFromID = func(id uuid.UUID) (fs.FileInfo, error) {
		return nil, nil
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		req := getRequest(nil, false, true)

		req.Form.Add("uuid", test.GetValidUUID())
		req.Form.Add("filename", tc.filename)

		api.RouteBlob(resWriter, req)

		assertZeroStatusCode(t, resWriter)
		assert.Equal(t, "public, max-age=604800, immutable", resWriter.Head.Get("Cache-Control"))
		assert.Equal(t, tc.expectedContentDisposition, resWriter.Head.Get("Content-Disposition"), tc.name)
	}
}

func TestSendTextFunctions(t *testing.T) {
	testcases := []struct {
		name                string
		testFunc            func(w http.ResponseWriter, req *http.Request)
		command             types.Command
		expectedContentType types.ContentType
	}{
		{
			name:                "RouteRoomCommandSetNick",
			testFunc:            api.RouteRoomCommandSetNick,
			command:             types.RoomCommandNick,
			expectedContentType: types.ContentTypeCmd,
		},
		{
			name:                "RouteRoomCommandNameRoom",
			testFunc:            api.RouteRoomCommandNameRoom,
			command:             types.RoomCommandNameRoom,
			expectedContentType: types.ContentTypeCmd,
		},
		{
			name:                "RouteRoomCommandPromote",
			testFunc:            api.RouteRoomCommandPromote,
			command:             types.RoomCommandPromote,
			expectedContentType: types.ContentTypeCmd,
		},
		{
			name:                "RouteRoomSendMessage",
			testFunc:            api.RouteRoomSendMessage,
			command:             "",
			expectedContentType: types.ContentTypeText,
		},
	}

	var (
		actualID         string
		actualMsgContent types.MessageContent
	)

	daemon.SendMessage = func(uuid string, content types.MessageContent) error {
		actualID = uuid
		actualMsgContent = content
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

		expectedMsgContent := types.MessageContent{
			Type: tc.expectedContentType,
			Data: []byte(expectedContent),
		}

		req, _ := http.NewRequest("", "", &reader)

		expectedID := "test id"
		req.Form = url.Values{}
		req.Form.Add("uuid", expectedID)

		tc.testFunc(resWriter, req)

		assertZeroStatusCode(t, resWriter)

		assert.Equal(t, expectedID, actualID, tc.name+": Uuid was modified")
		assert.Equal(t, expectedMsgContent, actualMsgContent)
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
			name:     "RouteRoomCommandPromote",
			testFunc: api.RouteRoomCommandPromote,
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
			req:               getRequest(nil, true, true),
			expectedErrorCode: http.StatusBadRequest,
		},
		{
			name:              "SendError",
			req:               getRequest([]string{"test content"}, false, true),
			expectedErrorCode: http.StatusInternalServerError,
		},
	}

	daemon.SendMessage = func(uuid string, content types.MessageContent) error {
		return test.GetTestError()
	}

	for _, tc := range testcases {
		for _, te := range testErrors {
			resWriter := mocks.GetMockResponseWriter()

			tc.testFunc(resWriter, te.req)

			assertErrorCode(t, resWriter, te.expectedErrorCode, tc.name+"-"+te.name)
		}
	}
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

func getRequest(body interface{}, readShouldError, marshalBody bool) *http.Request {
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

func assertApplicationJson(t *testing.T, resWriter *mocks.MockResponseWriter) {
	assert.Equal(t, "application/json", resWriter.Head.Get("Content-Type"), "Wrong value in header field Content-Type")
}
