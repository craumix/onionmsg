package blobmngr

import (
	"io"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
)

var (
	StreamTo      = streamTo
	MakeBlob      = makeBlob
	FileFromID    = fileFromID
	WriteIntoFile = writeIntoFile

	blobdir = "./"
)

func InitializeDir(dir string) error {
	err := os.Mkdir(dir, 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}

	blobdir = dir
	return nil
}

func GetRessource(id uuid.UUID) ([]byte, error) {
	return ioutil.ReadFile(blobPath(id))
}

func streamTo(id uuid.UUID, w io.Writer) error {
	file, err := FileFromID(id)
	if err != nil {
		return err
	}
	defer file.Close()

	io.Copy(w, file)
	/*
		buf := make([]byte, 4096)
		for n, err := file.Read(buf); n > 0 && err == nil; {
			_, err = w.Write(buf[:n])

			if err != nil {
				return err
			}
		}

		if err != nil {
			return err
		}
	*/

	return nil
}

func fileFromID(id uuid.UUID) (*os.File, error) {
	return os.OpenFile(blobPath(id), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
}

func StatFromID(id uuid.UUID) (fs.FileInfo, error) {
	return os.Stat(blobPath(id))
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

func SaveRessource(blob []byte) (uuid.UUID, error) {
	id := uuid.New()
	return id, ioutil.WriteFile(blobPath(id), blob, 0600)
}

func makeBlob() (uuid.UUID, error) {
	return SaveRessource(make([]byte, 0))
}

func RemoveBlob(id uuid.UUID) error {
	return os.Remove(blobPath(id))
}

func blobPath(id uuid.UUID) string {
	return blobdir + "/" + id.String() + ".blob"
}
