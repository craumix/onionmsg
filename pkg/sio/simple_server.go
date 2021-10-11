package sio

import (
	"net"
	"strconv"
)

//StartLocalServer is a wrapper for StartServer which uses "localhost" as the hostname.
//This makes the server only available for local connections.
func StartLocalServer(port int, clientHandler func(net.Conn), connErrHook func(error)) error {
	return StartServer(port, "localhost", clientHandler, connErrHook)
}

//StartServer opens a listener on the specified interface and port and passes connections
//to the provided handler, which is the startet as a new goroutine.
//The hostname can be omitted to listen on all interfaces.
func StartServer(port int, hostname string, clientHandler func(net.Conn), connErrHook func(error)) error {
	server, err := net.Listen("tcp", hostname+":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	defer server.Close()

	for {
		c, err := server.Accept()
		if err != nil {
			if connErrHook != nil {
				connErrHook(err)
			}
		}

		go clientHandler(c)
	}
}
