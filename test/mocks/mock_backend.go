package mocks

import (
	"github.com/craumix/onionmsg/internal/types"
	"github.com/craumix/onionmsg/pkg/blobmngr"
)

type MockBackend struct {
	BlobManager MockBlobManager

	TorInfoFunc                                   func() interface{}
	GetNotifierFunc                               func() types.Notifier
	GetContactIDsAsStringsFunc                    func() []string
	CreateAndRegisterNewContactIDFunc             func() (types.Identity, error)
	DeregisterAndRemoveContactIDByFingerprintFunc func(fingerprint string) error
	GetRoomRequestsFunc                           func() []*types.RoomRequest
	AcceptRoomRequestFunc                         func(id string) error
	RemoveRoomRequestByIDFunc                     func(toRemove string)
	GetRoomInfoByIDFunc                           func(roomId string) (*types.RoomInfo, error)
	GetInfoForAllRoomsFunc                        func() []*types.RoomInfo
	DeregisterAndDeleteRoomByIDFunc               func(roomID string) error
	CreateRoomFunc                                func(fingerprints []string) error
	SendMessageInRoomFunc                         func(roomID string, content types.MessageContent) error
	ListMessagesInRoomFunc                        func(roomID string, count int) ([]types.Message, error)
	AddNewPeerToRoomFunc                          func(roomID string, newPeerFingerprint string) error
}

func DefaultBackend() MockBackend {
	return MockBackend{}
}

func (m MockBackend) TorInfo() interface{} {
	return m.TorInfoFunc()
}

func (m MockBackend) GetNotifier() types.Notifier {
	return m.GetNotifierFunc()
}

func (m MockBackend) GetBlobManager() blobmngr.ManagesBlobs {
	return m.BlobManager
}

func (m MockBackend) GetContactIDsAsStrings() []string {
	return m.GetContactIDsAsStringsFunc()
}

func (m MockBackend) CreateAndRegisterNewContactID() (types.Identity, error) {
	return m.CreateAndRegisterNewContactIDFunc()
}

func (m MockBackend) DeregisterAndRemoveContactIDByFingerprint(fingerprint string) error {
	return m.DeregisterAndRemoveContactIDByFingerprintFunc(fingerprint)
}

func (m MockBackend) GetRoomRequests() []*types.RoomRequest {
	return m.GetRoomRequestsFunc()
}

func (m MockBackend) AcceptRoomRequest(id string) error {
	return m.AcceptRoomRequestFunc(id)
}

func (m MockBackend) RemoveRoomRequestByID(toRemove string) {
	m.RemoveRoomRequestByIDFunc(toRemove)
}

func (m MockBackend) GetRoomInfoByID(roomId string) (*types.RoomInfo, error) {
	return m.GetRoomInfoByIDFunc(roomId)
}

func (m MockBackend) GetInfoForAllRooms() []*types.RoomInfo {
	return m.GetInfoForAllRoomsFunc()
}

func (m MockBackend) DeregisterAndDeleteRoomByID(roomID string) error {
	return m.DeregisterAndDeleteRoomByIDFunc(roomID)
}

func (m MockBackend) CreateRoom(fingerprints []string) error {
	return m.CreateRoomFunc(fingerprints)
}

func (m MockBackend) SendMessageInRoom(roomID string, content types.MessageContent) error {
	return m.SendMessageInRoomFunc(roomID, content)
}

func (m MockBackend) ListMessagesInRoom(roomID string, count int) ([]types.Message, error) {
	return m.ListMessagesInRoomFunc(roomID, count)
}

func (m MockBackend) AddNewPeerToRoom(roomID string, newPeerFingerprint string) error {
	return m.AddNewPeerToRoomFunc(roomID, newPeerFingerprint)
}
