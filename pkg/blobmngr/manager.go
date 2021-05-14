package blobmngr

import (
	"io"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
)

var (
	blobdir = "./"
)

func Initialize(dir string) error {
	err := os.Mkdir(dir, 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}

	blobdir = dir
	return nil
}

func GetRessource(id uuid.UUID) ([]byte, error) {
	return ioutil.ReadFile(resolveResPath(id))
}

func StreamTo(id uuid.UUID, w io.Writer) error {
	file, err := FileFromID(id)
	if err != nil {
		return err
	}

	buf := make([]byte, 4096)
	for n, err := file.Read(buf); n > 0 && err != nil; {
		_, err = w.Write(buf[:n])
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	file.Close()

	return nil
}

func FileFromID(id uuid.UUID) (*os.File, error) {
	return os.OpenFile(resolveResPath(id), os.O_RDONLY, 0600)
}

func StatFromID(id uuid.UUID) (fs.FileInfo, error) {
	return os.Stat(resolveResPath(id))
}

func SaveRessource(blob []byte) (uuid.UUID, error) {
	id := uuid.New()
	return id, ioutil.WriteFile(resolveResPath(id), blob, 0600)
}

func MakeBlob() (uuid.UUID, error) {
	return SaveRessource(make([]byte, 0))
}

func resolveResPath(id uuid.UUID) string {
	return blobdir + "/" + id.String() + ".blob"
}
