package api_test

import (
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/craumix/onionmsg/internal/api"
	"github.com/craumix/onionmsg/internal/types"
	"github.com/craumix/onionmsg/test"
	"github.com/craumix/onionmsg/test/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"testing"
)

func defaultConf() Config {
	return Config{
		UseUnixSocket: false,
		PortGroup:     types.NewPortGroup(0),
	}
}

func TestRouteStatus(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	apiT := NewAPI(defaultConf(), nil)

	apiT.RouteStatus(resWriter, nil)

	assertZeroStatusCode(t, resWriter)
	assertApplicationJson(t, resWriter)
	assert.Equal(t, "{\"status\":\"ok\"}", string(resWriter.WriteInput[0]))
}

func TestRouteTorLog(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	backend := mocks.DefaultBackend()

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

	backend.TorInfoFunc = func() interface{} {
		return expected
	}

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteTorInfo(resWriter, nil)

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

	backend := mocks.DefaultBackend()

	expected := []string{"Contact1"}

	backend.GetContactIDsAsStringsFunc = func() []string {
		return expected
	}

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteContactList(resWriter, nil)

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

	backend := mocks.DefaultBackend()

	backend.GetInfoForAllRoomsFunc = func() []*types.RoomInfo {
		return expected
	}

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRoomList(resWriter, nil)

	var actual []*types.RoomInfo
	json.Unmarshal(resWriter.WriteInput[0], &actual)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "RoomInfo was modified")
}

func TestRouteRoomCreate(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	backend := mocks.DefaultBackend()

	var actual []string

	backend.CreateRoomFunc = func(fingerprints []string) error {
		actual = fingerprints
		return nil
	}

	expected := []string{"id1", "id2"}

	req := getRequest(expected, false, true)

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRoomCreate(resWriter, req)

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

	backend := mocks.DefaultBackend()

	backend.CreateRoomFunc = func(fingerprints []string) error {
		return test.GetTestError()
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()
		apiT := NewAPI(defaultConf(), backend)

		apiT.RouteRoomCreate(resWriter, tc.req)

		assertErrorCode(t, resWriter, tc.expectedErrorCode, tc.name)
	}
}

func TestDeleteRoom(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	var actual string

	backend := mocks.DefaultBackend()

	backend.DeregisterAndDeleteRoomByIDFunc = func(uuid string) error {
		actual = uuid
		return nil
	}

	req := getRequest(nil, false, true)

	expected := "test id"
	req.Form.Add("uuid", expected)

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRoomDelete(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "Uuid was modified")
}

func TestDeleteRoomError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	backend := mocks.DefaultBackend()

	backend.DeregisterAndDeleteRoomByIDFunc = func(uuid string) error {
		return test.GetTestError()
	}

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRoomDelete(resWriter, getRequest(nil, false, true))

	assertErrorCode(t, resWriter, http.StatusInternalServerError)
}

func TestRouteContactCreate(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	expected := types.NewContactIdentity()

	backend := mocks.DefaultBackend()

	backend.CreateAndRegisterNewContactIDFunc = func() (types.ContactIdentity, error) {
		return expected, nil
	}

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteContactCreate(resWriter, nil)

	assertZeroStatusCode(t, resWriter)
	assertApplicationJson(t, resWriter)
	assert.Equal(t, string(resWriter.WriteInput[0]), fmt.Sprintf("{\"id\":\"%s\"}", expected.Fingerprint()), "Uuid was modified")
}

func TestRouteContactCreateError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	backend := mocks.DefaultBackend()

	backend.CreateAndRegisterNewContactIDFunc = func() (types.ContactIdentity, error) {
		return types.ContactIdentity{}, test.GetTestError()
	}

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteContactCreate(resWriter, nil)

	assertErrorCode(t, resWriter, http.StatusInternalServerError)
}

func TestRouteContactDelete(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	backend := mocks.DefaultBackend()

	var actual string

	backend.DeregisterAndRemoveContactIDByFingerprintFunc = func(fingerprint string) error {
		actual = fingerprint
		return nil
	}

	req := getRequest(nil, false, true)

	expected := "test id"
	req.Form.Add("fingerprint", expected)

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteContactDelete(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual, "Uuid was modified")
}

func TestRouteContactDeleteNoID(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	backend := mocks.DefaultBackend()

	called := false

	backend.DeregisterAndRemoveContactIDByFingerprintFunc = func(fingerprint string) error {
		called = true
		return nil
	}

	req := getRequest(nil, false, true)

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteContactDelete(resWriter, req)

	assertErrorCode(t, resWriter, http.StatusBadRequest)
	assert.False(t, called, "Delete contact got called with missing id field!")
}

func TestRouteContactDeleteError(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	backend := mocks.DefaultBackend()

	backend.DeregisterAndRemoveContactIDByFingerprintFunc = func(fingerprint string) error {
		return test.GetTestError()
	}

	req := getRequest(nil, false, true)

	req.Form.Add("fingerprint", "test id")

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteContactDelete(resWriter, req)

	assertErrorCode(t, resWriter, http.StatusInternalServerError)
}

func TestRouteRoomCommandUseradd(t *testing.T) {
	resWriter := mocks.GetMockResponseWriter()

	var actualID, actualFp string

	backend := mocks.DefaultBackend()

	backend.AddNewPeerToRoomFunc = func(roomID string, fingerprint string) error {
		actualID = roomID
		actualFp = fingerprint
		return nil
	}

	expectedFp, expectedID := "test content", test.GetValidUUID()

	req := getRequest(expectedFp, false, false)

	req.Form.Add("uuid", expectedID)

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRoomCommandUseradd(resWriter, req)

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
			name:              "AddNewPeerToRoom error",
			req:               getRequest([]string{"test content"}, false, true),
			uuid:              test.GetValidUUID(),
			expectedErrorCode: http.StatusInternalServerError,
		},
	}

	backend := mocks.DefaultBackend()

	backend.AddNewPeerToRoomFunc = func(roomID string, fingerprint string) error {
		return test.GetTestError()
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		tc.req.Form.Add("uuid", tc.uuid)

		apiT := NewAPI(defaultConf(), backend)

		apiT.RouteRoomCommandUseradd(resWriter, tc.req)

		assertErrorCode(t, resWriter, tc.expectedErrorCode, tc.name)
	}

}

func TestRoomSendFile(t *testing.T) {

	manager := mocks.DefaultBlobManager()

	newBlobId := uuid.New()
	manager.MakeBlobFunc = func() (uuid.UUID, error) {
		return newBlobId, nil
	}

	writtenInto := false
	manager.WriteIntoBlobFunc = func(from io.Reader, blobID uuid.UUID) error {
		writtenInto = true
		return nil
	}

	manager.StatFromIDFunc = func(id uuid.UUID) (fs.FileInfo, error) {
		return mocks.MockFileInfo{}, nil
	}

	backend := mocks.DefaultBackend()
	backend.BlobManager = manager

	var (
		actualID         string
		actualMsgContent types.MessageContent
	)
	backend.SendMessageInRoomFunc = func(uuid string, msgContent types.MessageContent) error {
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
			Size: 42,
		},
		Data: nil,
	}

	req.Header.Set(FilenameHeader, expectedMsgContent.Blob.Name)
	req.Header.Set(MimetypeHeader, expectedMsgContent.Blob.Type)
	req.Header.Set("Content-Length", "69")

	resWriter := mocks.GetMockResponseWriter()

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRoomSendFile(resWriter, req)

	// TODO check if the file pointer is the same

	assertZeroStatusCode(t, resWriter)

	assert.Truef(t, writtenInto, "Generated Blob wasn't written into")
	//assert.Equal(t, newBlobId.String(), actualFileId.String(), "FileFromID was called with a different id than generated")
	assert.Equal(t, expectedID, actualID, "SendMessageInRoom didn't get the Id from the request")
	assert.Equal(t, expectedMsgContent, actualMsgContent)
}

func TestRoomSendFileErrors(t *testing.T) {
	testcases := []struct {
		name             string
		expectedErrCode  int
		fileLength       string
		MakeBlobErr      error
		WriteIntoBlobErr error
		SendErr          error
	}{
		{
			name:            "MakeBlobError",
			expectedErrCode: http.StatusInternalServerError,
			MakeBlobErr:     test.GetTestError(),
		},
		{
			name:             "WriteIntoBlobErr",
			expectedErrCode:  http.StatusInternalServerError,
			WriteIntoBlobErr: test.GetTestError(),
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

		manager := mocks.DefaultBlobManager()

		manager.MakeBlobFunc = func() (uuid.UUID, error) {
			return uuid.UUID{}, tc.MakeBlobErr
		}

		manager.WriteIntoBlobFunc = func(from io.Reader, blobID uuid.UUID) error {
			return tc.WriteIntoBlobErr
		}

		manager.StatFromIDFunc = func(id uuid.UUID) (fs.FileInfo, error) {
			return mocks.MockFileInfo{}, nil
		}

		backend := mocks.DefaultBackend()
		backend.BlobManager = manager

		backend.SendMessageInRoomFunc = func(uuid string, content types.MessageContent) error {
			return tc.SendErr
		}

		if tc.fileLength == "" {
			tc.fileLength = "42"
		}

		req := getRequest(nil, false, true)
		req.Header.Set("Content-Length", tc.fileLength)

		apiT := NewAPI(defaultConf(), backend)

		apiT.RouteRoomSendFile(resWriter, req)

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
			name:            "ListMessagesInRoom error",
			ListMessagesErr: test.GetTestError(),
			expectedErrCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		backend := mocks.DefaultBackend()

		var actualID string
		backend.ListMessagesInRoomFunc = func(uuid string, count int) ([]types.Message, error) {
			actualID = uuid
			return nil, tc.ListMessagesErr
		}

		req := getRequest(nil, false, true)

		req.Form.Add("count", tc.expectedCount)

		apiT := NewAPI(defaultConf(), backend)

		apiT.RouteRoomMessages(resWriter, req)

		assertErrorCode(t, resWriter, tc.expectedErrCode, tc.name)
		assert.Equal(t, tc.expectedID, actualID, tc.name+": Uuid was modified")
	}
}

func TestRouteBlob(t *testing.T) {

	manager := mocks.DefaultBlobManager()

	var actualID string
	manager.StreamToFunc = func(id uuid.UUID, w io.Writer) error {
		actualID = id.String()
		return nil
	}

	manager.StatFromIDFunc = func(id uuid.UUID) (fs.FileInfo, error) {
		return mocks.MockFileInfo{}, nil
	}

	req := getRequest(nil, false, true)

	expectedID := test.GetValidUUID()
	req.Form.Add("uuid", expectedID)

	resWriter := mocks.GetMockResponseWriter()

	backend := mocks.DefaultBackend()
	backend.BlobManager = manager

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteBlob(resWriter, req)

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

	manager := mocks.DefaultBlobManager()

	manager.StatFromIDFunc = func(id uuid.UUID) (fs.FileInfo, error) {
		return mocks.MockFileInfo{}, nil
	}

	for _, tc := range testcases {
		manager.StreamToFunc = func(id uuid.UUID, w io.Writer) error {
			return tc.StreamToErr
		}

		req := getRequest(nil, false, true)
		req.Form.Add("uuid", tc.id)

		resWriter := mocks.GetMockResponseWriter()

		backend := mocks.DefaultBackend()
		backend.BlobManager = manager

		apiT := NewAPI(defaultConf(), backend)

		apiT.RouteBlob(resWriter, req)

		assertErrorCode(t, resWriter, tc.expectedErrCode, tc.name)
	}
}

func TestRouteBlobBlobNotFoundError(t *testing.T) {
	manager := mocks.DefaultBlobManager()

	called := false
	manager.StreamToFunc = func(id uuid.UUID, w io.Writer) error {
		called = true
		return nil
	}

	manager.StatFromIDFunc = func(id uuid.UUID) (fs.FileInfo, error) {
		return mocks.MockFileInfo{}, os.ErrNotExist
	}

	req := getRequest(nil, false, true)

	expectedID := test.GetValidUUID()
	req.Form.Add("uuid", expectedID)

	resWriter := mocks.GetMockResponseWriter()

	backend := mocks.DefaultBackend()
	backend.BlobManager = manager

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteBlob(resWriter, req)

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

	manager := mocks.DefaultBlobManager()

	manager.StreamToFunc = func(id uuid.UUID, w io.Writer) error {
		return nil
	}

	manager.StatFromIDFunc = func(id uuid.UUID) (fs.FileInfo, error) {
		return mocks.MockFileInfo{}, nil
	}

	backend := mocks.DefaultBackend()
	backend.BlobManager = manager

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()
		apiT := NewAPI(defaultConf(), backend)

		req := getRequest(nil, false, true)

		req.Form.Add("uuid", test.GetValidUUID())
		req.Form.Add("filename", tc.filename)

		apiT.RouteBlob(resWriter, req)

		assertZeroStatusCode(t, resWriter)
		assert.Equal(t, "public, max-age=604800, immutable", resWriter.Head.Get("Cache-Control"))
		assert.Equal(t, tc.expectedContentDisposition, resWriter.Head.Get("Content-Disposition"), tc.name)
	}
}

func TestSendTextFunctions(t *testing.T) {
	var (
		actualID         string
		actualMsgContent types.MessageContent
	)

	backend := mocks.DefaultBackend()

	backend.SendMessageInRoomFunc = func(uuid string, content types.MessageContent) error {
		actualID = uuid
		actualMsgContent = content
		return nil
	}

	apiT := NewAPI(defaultConf(), backend)

	testcases := []struct {
		name                string
		testFunc            func(w http.ResponseWriter, req *http.Request)
		command             types.Command
		expectedContentType types.ContentType
	}{
		{
			name:                "RouteRoomCommandSetNick",
			testFunc:            apiT.RouteRoomCommandSetNick,
			command:             types.RoomCommandNick,
			expectedContentType: types.ContentTypeCmd,
		},
		{
			name:                "RouteRoomCommandNameRoom",
			testFunc:            apiT.RouteRoomCommandNameRoom,
			command:             types.RoomCommandNameRoom,
			expectedContentType: types.ContentTypeCmd,
		},
		{
			name:                "RouteRoomCommandPromote",
			testFunc:            apiT.RouteRoomCommandPromote,
			command:             types.RoomCommandPromote,
			expectedContentType: types.ContentTypeCmd,
		},
		{
			name:                "RoomCommandRemovePeer",
			testFunc:            apiT.RouteRoomCommandRemovePeer,
			command:             types.RoomCommandRemovePeer,
			expectedContentType: types.ContentTypeCmd,
		},
		{
			name:                "RouteRoomSendMessage",
			testFunc:            apiT.RouteRoomSendMessage,
			command:             "",
			expectedContentType: types.ContentTypeText,
		},
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
	backend := mocks.DefaultBackend()

	backend.SendMessageInRoomFunc = func(uuid string, content types.MessageContent) error {
		return test.GetTestError()
	}

	apiT := NewAPI(defaultConf(), backend)

	testcases := []struct {
		name     string
		testFunc func(w http.ResponseWriter, req *http.Request)
	}{
		{
			name:     "RouteRoomCommandSetNick",
			testFunc: apiT.RouteRoomCommandSetNick,
		},
		{
			name:     "RouteRoomCommandNameRoom",
			testFunc: apiT.RouteRoomCommandNameRoom,
		},
		{
			name:     "RouteRoomCommandPromote",
			testFunc: apiT.RouteRoomCommandPromote,
		},
		{
			name:     "RouteRoomSendMessage",
			testFunc: apiT.RouteRoomSendMessage,
		},
		{
			name:     "RouteRoomCommandRemovePeer",
			testFunc: apiT.RouteRoomCommandRemovePeer,
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

	for _, tc := range testcases {
		for _, te := range testErrors {
			resWriter := mocks.GetMockResponseWriter()

			tc.testFunc(resWriter, te.req)

			assertErrorCode(t, resWriter, te.expectedErrorCode, tc.name+"-"+te.name)
		}
	}
}

func TestRouteRequestList(t *testing.T) {
	backend := mocks.DefaultBackend()

	expected := []types.RoomRequest{
		{
			Room:           types.Room{},
			ViaFingerprint: "test-fingerprint",
			ID:             uuid.UUID{},
		},
	}

	backend.GetRoomRequestsFunc = func() []*types.RoomRequest {
		var res []*types.RoomRequest
		for _, rr := range expected {
			res = append(res, &rr)
		}
		return res
	}

	req := getRequest(nil, false, true)

	resWriter := mocks.GetMockResponseWriter()

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRequestList(resWriter, req)

	var actual []types.RoomRequest
	json.Unmarshal(resWriter.WriteInput[0], &actual)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expected, actual)
}

func TestRouteRequestAccept(t *testing.T) {
	backend := mocks.DefaultBackend()

	actualID := ""
	backend.AcceptRoomRequestFunc = func(id string) error {
		actualID = id
		return nil
	}

	req := getRequest(nil, false, true)

	expectedID := test.GetValidUUID()
	req.Form.Add("uuid", expectedID)

	resWriter := mocks.GetMockResponseWriter()

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRequestAccept(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expectedID, actualID)
}

func TestMalformedUuidErrors(t *testing.T) {
	apiT := NewAPI(defaultConf(), mocks.DefaultBackend())

	testcases := []struct {
		name     string
		testFunc func(w http.ResponseWriter, req *http.Request)
	}{
		{
			name:     "RouteRequestAccept",
			testFunc: apiT.RouteRequestAccept,
		},
		{
			name:     "RouteRequestDelete",
			testFunc: apiT.RouteRequestDelete,
		},
		{
			name:     "RouteRoomInfo",
			testFunc: apiT.RouteRoomInfo,
		},
	}

	for _, tc := range testcases {
		resWriter := mocks.GetMockResponseWriter()

		req := getRequest(nil, true, true)
		req.Form.Add("uuid", "MalformedUUID")

		tc.testFunc(resWriter, req)

		assertErrorCode(t, resWriter, http.StatusBadRequest, tc.name)
	}
}

func TestRouteRequestAcceptError(t *testing.T) {
	testcases := []struct {
		name       string
		id         string
		errCode    int
		backendErr error
	}{
		{
			name:       "RoomId not accepted",
			id:         test.GetValidUUID(),
			errCode:    http.StatusInternalServerError,
			backendErr: test.GetTestError(),
		},
	}

	for _, tc := range testcases {
		backend := mocks.DefaultBackend()

		backend.AcceptRoomRequestFunc = func(id string) error {
			return tc.backendErr
		}

		req := getRequest(nil, false, true)

		req.Form.Add("uuid", tc.id)

		resWriter := mocks.GetMockResponseWriter()

		apiT := NewAPI(defaultConf(), backend)

		apiT.RouteRequestAccept(resWriter, req)

		assertErrorCode(t, resWriter, tc.errCode)
	}
}

func TestRouteRequestDelete(t *testing.T) {
	backend := mocks.DefaultBackend()

	var actualID string
	backend.RemoveRoomRequestByIDFunc = func(toRemove string) {
		actualID = toRemove
	}

	req := getRequest(nil, false, true)

	expectedID := test.GetValidUUID()
	req.Form.Add("uuid", expectedID)

	resWriter := mocks.GetMockResponseWriter()

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRequestDelete(resWriter, req)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expectedID, actualID, "Uuid was modified")
}

func TestRouteRoomInfo(t *testing.T) {
	backend := mocks.DefaultBackend()

	expectedInfo := types.RoomInfo{
		Self:   "test-self",
		Peers:  []string{"a", "b"},
		ID:     uuid.UUID{},
		Name:   "test-name",
		Nicks:  nil,
		Admins: nil,
	}

	var actualID string
	backend.GetRoomInfoByIDFunc = func(roomId string) (*types.RoomInfo, error) {
		actualID = roomId
		return &expectedInfo, nil
	}

	req := getRequest(nil, false, true)

	expectedID := test.GetValidUUID()
	req.Form.Add("uuid", expectedID)

	resWriter := mocks.GetMockResponseWriter()

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRoomInfo(resWriter, req)

	var actualInfo types.RoomInfo
	json.Unmarshal(resWriter.WriteInput[0], &actualInfo)

	assertZeroStatusCode(t, resWriter)
	assert.Equal(t, expectedID, actualID, "Uuid was modified")
	assert.Equal(t, expectedInfo, actualInfo, "Wrong Info")
}

func TestTestRouteRoomInfoError(t *testing.T) {
	backend := mocks.DefaultBackend()

	backend.GetRoomInfoByIDFunc = func(roomId string) (*types.RoomInfo, error) {
		return nil, test.GetTestError()
	}

	req := getRequest(nil, false, true)

	req.Form.Add("uuid", test.GetValidUUID())

	resWriter := mocks.GetMockResponseWriter()

	apiT := NewAPI(defaultConf(), backend)

	apiT.RouteRoomInfo(resWriter, req)

	assertErrorCode(t, resWriter, http.StatusNotFound)
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
