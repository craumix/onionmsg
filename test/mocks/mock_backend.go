package mocks

import (
	"github.com/craumix/onionmsg/internal/types"
	"github.com/google/uuid"
)

type MockBackend struct {
	MockBlobManager

	TorInfoFunc            func() interface{}
	GetNotifierFunc        func() types.Notifier
	GetContactIDsFunc      func() []types.ContactIdentity
	CreateContactIDFunc    func() (types.ContactIdentity, error)
	DeleteContactIDFunc    func(fingerprint types.Fingerprint) error
	GetRoomInfoFunc        func(roomId uuid.UUID) (*types.RoomInfo, error)
	GetInfoForAllRoomsFunc func() []*types.RoomInfo
	DeleteRoomFunc         func(roomID uuid.UUID) error
	CreateRoomFunc         func(fingerprints []string) error
	SendMessageInRoomFunc  func(roomID uuid.UUID, content types.MessageContent) error
	ListMessagesInRoomFunc func(roomID uuid.UUID, count int) ([]types.Message, error)
	AddNewPeerToRoomFunc   func(roomID uuid.UUID, newPeer types.Fingerprint) error
	GetRoomRequestsFunc    func() []*types.RoomRequest
	AcceptRoomRequestFunc  func(roomReqID uuid.UUID) error
	DeleteRoomRequestFunc  func(roomReqID uuid.UUID)
}

func (m MockBackend) AddObserver(newObserver types.Observer) {
	//TODO implement me
	panic("AddObserver in MockBackend not implemented!")
}

func (m MockBackend) TorInfo() interface{} {
	return m.TorInfoFunc()
}

func (m MockBackend) GetNotifier() types.Notifier {
	return m.GetNotifierFunc()
}

func (m MockBackend) GetContactIDs() []types.ContactIdentity {
	return m.GetContactIDsFunc()
}

func (m MockBackend) CreateContactID() (types.ContactIdentity, error) {
	return m.CreateContactIDFunc()
}

func (m MockBackend) DeleteContactID(fingerprint types.Fingerprint) error {
	return m.DeleteContactIDFunc(fingerprint)
}

func (m MockBackend) GetRoomInfo(roomId uuid.UUID) (*types.RoomInfo, error) {
	return m.GetRoomInfoFunc(roomId)
}

func (m MockBackend) GetInfoForAllRooms() []*types.RoomInfo {
	return m.GetInfoForAllRoomsFunc()
}

func (m MockBackend) DeleteRoom(roomID uuid.UUID) error {
	return m.DeleteRoomFunc(roomID)
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

func (m MockBackend) DeleteRoomRequest(roomReqID uuid.UUID) {
	m.DeleteRoomRequestFunc(roomReqID)
}

func DefaultBackend() MockBackend {
	return MockBackend{}
}
