package daemon

import (
	"os"
	"strconv"
	"strings"
)

type TorStringWriter struct {
	OnWrite func(string)
}

func (w TorStringWriter) Write(p []byte) (int, error) {
	if w.OnWrite != nil {
		lines := strings.Split(string(p), "\n")
		lines = lines[:len(lines)-1]
		for _, v := range lines {
			w.OnWrite(v)
		}
	}
	return len(p), nil
}

func writePIDFile(path string) error {
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	pid := os.Getpid()
	_, err = file.Write([]byte(strconv.Itoa(pid)))
	if err != nil {
		return err
	}

	return err
}
