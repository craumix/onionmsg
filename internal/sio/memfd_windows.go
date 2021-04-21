package sio

import (
	"fmt"
	"runtime"
)

func CreateMemFD(name string) (path string, err error) {
	return nil, fmt.Errorf("creating a memfd is not aviable on %s", runtime.GOOS)
}