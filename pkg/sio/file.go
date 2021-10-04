package sio

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/klauspost/compress/zstd"
)

//SaveDataCompressed marshals the provided struct, zstd compresses it and the writes it to the file specified by the provided path.
func SaveDataCompressed(datafile string, src interface{}) error {
	file, err := os.OpenFile(datafile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}

	enc, _ := zstd.NewWriter(file)
	comp := enc.EncodeAll(raw, make([]byte, 0))

	_, err = file.Write(comp)
	if err != nil {
		return err
	}

	//log.Printf("Written %d compressed bytes, was %d (%.2f%%)\n", len(comp), len(raw), (float64(len(comp))/float64(len(raw)))*100)

	return nil
}

//LoadCompressedData loads the file specified by the path, zstd decompresses it and the tries to unmarshal it into the provided struct.
func LoadCompressedData(datafile string, dest interface{}) error {
	file, err := os.OpenFile(datafile, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	comp, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	dec, _ := zstd.NewReader(nil)
	raw, _ := dec.DecodeAll(comp, nil)

	json.Unmarshal(raw, dest)

	//log.Printf("Decoded %d bytes from file contents\n", len(raw))

	return nil
}
