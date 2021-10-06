package sio

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
)

//CreateUnixSocket tries to create a unix socket with the specified name.
//Tries to use the path from env XDG_RUNTIME_DIR.
//If XDG_RUNTIME_DIR is not set, the socket is created in /tmp.
func CreateUnixSocket(name string) (socket net.Listener, err error) {
	if runtime.GOOS == "linux" {
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeDir == "" {
			runtimeDir = "/tmp"
			//log.Debugf("Unable to determine Env XDG_RUNTIME_DIR, using %s\n", runtimeDir)
		}

		socketPath := runtimeDir + "/" + name

		if _, ferr := os.Stat(socketPath); ferr == nil {
			//log.Debugf("Unix socket already exists, removing")
			err = os.Remove(socketPath)
			if err != nil {
				return
			}
		}

		socket, err = net.Listen("unix", socketPath)
		return
	}

	err = fmt.Errorf("cannot use unix socket on %s", runtime.GOOS)
	return
}

//CreateTCPSocket creates a socket listening on the loopback interface with the specified port.
func CreateTCPSocket(port int) (socket net.Listener, err error) {
	address := "127.0.0.1:" + strconv.Itoa(port)
	socket, err = net.Listen("tcp", address)
	return
}
