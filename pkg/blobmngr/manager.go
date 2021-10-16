package blobmngr

import (
	"io"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
)

type BlobManager struct {
	dir string
}

func NewBlobManager(dir string) BlobManager {
	return BlobManager{
		dir: dir,
	}
}

func (bm BlobManager) CreateDirIfNotExists() error {
	err := os.Mkdir(bm.dir, 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}

	return nil
}

func (bm BlobManager) GetResource(id uuid.UUID) ([]byte, error) {
	return ioutil.ReadFile(bm.blobPath(id))
}

func (bm BlobManager) StreamTo(id uuid.UUID, w io.Writer) error {
	file, err := bm.FileFromID(id)
	if err != nil {
		return err
	}
	defer file.Close()

	io.Copy(w, file)

	return nil
}

func (bm BlobManager) FileFromID(id uuid.UUID) (*os.File, error) {
	return os.OpenFile(bm.blobPath(id), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
}

func (bm BlobManager) StatFromID(id uuid.UUID) (fs.FileInfo, error) {
	return os.Stat(bm.blobPath(id))
}

func writeIntoFile(from io.Reader, to *os.File) error {
	buf := make([]byte, 4096)
	var n int
	for {
		n, _ = from.Read(buf)
		if n == 0 {
			break
		}

		_, err := to.Write(buf[:n])
		if err != nil {
			return err
		}
	}

	return nil
}

func (bm BlobManager) WriteIntoBlob(from io.Reader, blobID uuid.UUID) error {
	file, err := bm.FileFromID(blobID)
	if err != nil {
		return err
	}

	err = writeIntoFile(from, file)
	if err != nil {
		return err
	}

	return nil
}

func (bm BlobManager) SaveResource(blob []byte) (uuid.UUID, error) {
	id := uuid.New()
	return id, ioutil.WriteFile(bm.blobPath(id), blob, 0600)
}

func (bm BlobManager) MakeBlob() (uuid.UUID, error) {
	return bm.SaveResource(make([]byte, 0))
}

func (bm BlobManager) RemoveBlob(id uuid.UUID) error {
	return os.Remove(bm.blobPath(id))
}

func (bm BlobManager) blobPath(id uuid.UUID) string {
	return bm.dir + "/" + id.String() + ".blob"
}
