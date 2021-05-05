package blobmngr

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
)

var (
	blobdir = "./"
)

func Initialize(dir string) error {
	err := os.Mkdir(dir, 0600)
	if err != nil && !os.IsExist(err) {
		return err
	}

	blobdir = dir
	return nil
}

func GetRessource(id uuid.UUID) ([]byte, error) {
	return ioutil.ReadFile(resolveResPath(id))
}

func Stream(id uuid.UUID, w io.Writer) error {
	file, err := os.OpenFile(resolveResPath(id), os.O_RDONLY, 0600)
	if err != nil {
		return err
	}

	buf := make([]byte, 4096)
	for n, err := file.Read(buf); n > 0 && err != nil; {
		_, err := w.Write(buf[:n])
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

func SaveRessource(blob []byte) (uuid.UUID, error) {
	id := uuid.New()
	return id, ioutil.WriteFile(resolveResPath(id), blob, 0600)
}

func resolveResPath(id uuid.UUID) string {
	return blobdir + "/" + id.String() + ".blob"
}
