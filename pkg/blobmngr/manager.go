package blobmngr

import (
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

func SaveRessource(blob []byte) (uuid.UUID, error) {
	id := uuid.New()
	return id, ioutil.WriteFile(resolveResPath(id), blob, 0600)
}

func resolveResPath(id uuid.UUID) string {
	return blobdir + "/" + id.String() + ".blob"
}
