package mocks

import (
	"github.com/craumix/onionmsg/internal/types"
	"github.com/craumix/onionmsg/pkg/blobmngr"
	"github.com/google/uuid"
)

type MockBackend struct {
	BlobManager MockBlobManager

	TorInfoFunc                       func() interface{}
	GetNotifierFunc                   func() types.Notifier
	GetContactIDsAsStringsFunc        func() []string
	CreateAndRegisterNewContactIDFunc func() (types.ContactIdentity, error)
	DeregisterAndRemoveContactIDFunc  func(fingerprint types.Fingerprint) error
	GetRoomInfoFunc                   func(roomId uuid.UUID) (*types.RoomInfo, error)
	GetInfoForAllRoomsFunc            func() []*types.RoomInfo
	DeregisterAndDeleteRoomFunc       func(roomID uuid.UUID) error
	CreateRoomFunc                    func(fingerprints []string) error
	SendMessageInRoomFunc             func(roomID uuid.UUID, content types.MessageContent) error
	ListMessagesInRoomFunc            func(roomID uuid.UUID, count int) ([]types.Message, error)
	AddNewPeerToRoomFunc              func(roomID uuid.UUID, newPeer types.Fingerprint) error
	GetRoomRequestsFunc               func() []*types.RoomRequest
	AcceptRoomRequestFunc             func(roomReqID uuid.UUID) error
	RemoveRoomRequestFunc             func(roomReqID uuid.UUID)
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

func (m MockBackend) CreateAndRegisterNewContactID() (types.ContactIdentity, error) {
	return m.CreateAndRegisterNewContactIDFunc()
}

func (m MockBackend) DeregisterAndRemoveContactID(fingerprint types.Fingerprint) error {
	return m.DeregisterAndRemoveContactIDFunc(fingerprint)
}

func (m MockBackend) GetRoomInfo(roomId uuid.UUID) (*types.RoomInfo, error) {
	return m.GetRoomInfoFunc(roomId)
}

func (m MockBackend) GetInfoForAllRooms() []*types.RoomInfo {
	return m.GetInfoForAllRoomsFunc()
}

func (m MockBackend) DeregisterAndDeleteRoom(roomID uuid.UUID) error {
	return m.DeregisterAndDeleteRoomFunc(roomID)
}

func (m MockBackend) CreateRoom(fingerprints []string) error {
	return m.CreateRoomFunc(fingerprints)
}

func (m MockBackend) SendMessageInRoom(roomID uuid.UUID, content types.MessageContent) error {
	return m.SendMessageInRoomFunc(roomID, content)
}

func (m MockBackend) ListMessagesInRoom(roomID uuid.UUID, count int) ([]types.Message, error) {
	return m.ListMessagesInRoomFunc(roomID, count)
}

func (m MockBackend) AddNewPeerToRoom(roomID uuid.UUID, newPeer types.Fingerprint) error {
	return m.AddNewPeerToRoomFunc(roomID, newPeer)
}

func (m MockBackend) GetRoomRequests() []*types.RoomRequest {
	return m.GetRoomRequestsFunc()
}

func (m MockBackend) AcceptRoomRequest(roomReqID uuid.UUID) error {
	return m.AcceptRoomRequestFunc(roomReqID)
}

func (m MockBackend) RemoveRoomRequest(roomReqID uuid.UUID) {
	m.RemoveRoomRequestFunc(roomReqID)
}

func DefaultBackend() MockBackend {
	return MockBackend{}
}
