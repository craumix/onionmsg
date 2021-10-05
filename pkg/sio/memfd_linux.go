package sio

import (
	"fmt"
	"strconv"

	"golang.org/x/sys/unix"
)

//CreateMemFD creates an anonymous file and then returns the path for the created file.
func CreateMemFD(name string) (path string, err error) {
	handle, err := unix.MemfdCreate(name, 0)
	if err != nil {
		return "", fmt.Errorf("unable to create Memfd \"%s\"  %s", name, err)
	}

	path = "/proc/self/fd/" + strconv.Itoa(handle)

	return path, nil
}
