package mocks

import (
	"github.com/google/uuid"
	"io"
	"io/fs"
	"os"
)

type MockBlobManager struct {
	CreateDirIfNotExistsFunc func() error
	GetResourceFunc          func(id uuid.UUID) ([]byte, error)
	StreamToFunc             func(id uuid.UUID, w io.Writer) error
	FileFromIDFunc           func(id uuid.UUID) (*os.File, error)
	StatFromIDFunc           func(id uuid.UUID) (fs.FileInfo, error)
	WriteIntoBlobFunc        func(from io.Reader, blobID uuid.UUID) error
	SaveResourceFunc         func(blob []byte) (uuid.UUID, error)
	MakeBlobFunc             func() (uuid.UUID, error)
	RemoveBlobFunc           func(id uuid.UUID) error
}

func DefaultBlobManager() MockBlobManager {
	return MockBlobManager{}
}

func (m MockBlobManager) CreateDirIfNotExists() error {
	return m.CreateDirIfNotExistsFunc()
}

func (m MockBlobManager) GetResource(id uuid.UUID) ([]byte, error) {
	return m.GetResourceFunc(id)
}

func (m MockBlobManager) StreamTo(id uuid.UUID, w io.Writer) error {
	return m.StreamToFunc(id, w)
}

func (m MockBlobManager) FileFromID(id uuid.UUID) (*os.File, error) {
	return m.FileFromIDFunc(id)
}

func (m MockBlobManager) StatFromID(id uuid.UUID) (fs.FileInfo, error) {
	return m.StatFromIDFunc(id)
}

func (m MockBlobManager) WriteIntoBlob(from io.Reader, blobID uuid.UUID) error {
	return m.WriteIntoBlobFunc(from, blobID)
}

func (m MockBlobManager) SaveResource(blob []byte) (uuid.UUID, error) {
	return m.SaveResourceFunc(blob)
}

func (m MockBlobManager) MakeBlob() (uuid.UUID, error) {
	return m.MakeBlobFunc()
}

func (m MockBlobManager) RemoveBlob(id uuid.UUID) error {
	return m.RemoveBlobFunc(id)
}
