package types

import (
	"github.com/google/uuid"
	"io"
	"io/fs"
	"os"
)

type RoomManager interface {
	GetRoomInfo(roomId uuid.UUID) (*RoomInfo, error)
	GetInfoForAllRooms() []*RoomInfo
	CreateRoom(fingerprints []string) error
	DeleteRoom(roomID uuid.UUID) error
	SendMessageInRoom(roomID uuid.UUID, content MessageContent) error
	ListMessagesInRoom(roomID uuid.UUID, count int) ([]Message, error)
	AddNewPeerToRoom(roomID uuid.UUID, newPeer Fingerprint) error
}

type RoomRequestManager interface {
	AcceptRoomRequest(roomReqID uuid.UUID) error
	DeleteRoomRequest(roomReqID uuid.UUID)
	GetRoomRequests() []*RoomRequest
}

type ContactManager interface {
	CreateContactID() (ContactIdentity, error)
	DeleteContactID(fingerprint Fingerprint) error
	GetContactIDs() []ContactIdentity
}

type BlobManager interface {
	CreateDirIfNotExists() error
	GetResource(id uuid.UUID) ([]byte, error)
	StreamTo(id uuid.UUID, w io.Writer) error
	FileFromID(id uuid.UUID) (*os.File, error)
	StatFromID(id uuid.UUID) (fs.FileInfo, error)
	WriteIntoBlob(from io.Reader, blobID uuid.UUID) error
	SaveResource(blob []byte) (uuid.UUID, error)
	MakeBlob() (uuid.UUID, error)
	RemoveBlob(id uuid.UUID) error
}
