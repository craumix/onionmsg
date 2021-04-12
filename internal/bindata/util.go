package bindata

import "os"

func WriteToFile(path, name string) (int, error) {
	file, err := os.OpenFile(path, os.O_WRONLY, 0700)
	if err != nil {
		return 0, err
	}

	torBinary, err := Asset("build/tor/tor")
	if err != nil {
		return 0, err
	}

	n, err := file.Write(torBinary)
	if err != nil {
		return 0, err
	}
	file.Close()

	return n, nil
}