package sio

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
)

func CreateUnixSocket(name string) (socket net.Listener, err error) {
	if runtime.GOOS == "linux" {
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeDir == "" {
			runtimeDir = "/tmp"
			log.Printf("Unable to determine Env XDG_RUNTIME_DIR, using %s\n", runtimeDir)
		}

		socketPath := runtimeDir + "/" + name
		log.Printf("Using unix socket with path %s\n", socketPath)

		if _, ferr := os.Stat(socketPath); ferr == nil {
			log.Printf("Unix socket already exists, removing")
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

func CreateTCPSocket(port int) (socket net.Listener, err error) {
	address := "127.0.0.1:" + strconv.Itoa(port)
	log.Printf("Starting socket on on %s\n", address)
	socket, err = net.Listen("tcp", address)
	return
}
