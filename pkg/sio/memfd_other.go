//go:build darwin || windows
// +build darwin windows

package sio

import (
	"fmt"
	"runtime"
)

func CreateMemFD(name string) (path string, err error) {
	return "", fmt.Errorf("creating a memfd is not aviable on %s", runtime.GOOS)
}
