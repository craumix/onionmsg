package sio

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/DataDog/zstd"
)

func SaveDataCompressed(datafile string, src interface{}) error {
	file, err := os.OpenFile(datafile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	raw, _ := json.Marshal(src)
	comp, _ := zstd.Compress(nil, raw)

	_, err = file.Write(comp)
	if err != nil {
		return err
	}
	
	log.Printf("Written %d compressed bytes, was %d (%.2f%%)\n", len(comp), len(raw), (float64(len(comp)) / float64(len(raw))) * 100)

	return nil
}

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

	raw, _:= zstd.Decompress(nil, comp)
	
	json.Unmarshal(raw, dest)

	log.Printf("Decoded %d bytes from file contents\n", len(raw))

	return nil
}