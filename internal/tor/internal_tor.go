// +build internalTor

package tor

import (
	"fmt"
	"github.com/craumix/onionmsg/pkg/sio"
	"log"
	"os"
	"runtime"
	"strconv"
)

var (
	torBinMemFD string
)

func getExePath() (string, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("cannot use internal tor binary on platform \"%s\"", runtime.GOOS)
	}

	return binToMem()
}

func binToMem() (string, error) {
	if torBinMemFD != "" {
		return torBinMemFD, nil
	}

	memfd, err := sio.CreateMemFD("tormemfd")
	if err != nil {
		return "", err
	}

	n, err := WriteToFile(memfd, "../../third_party/tor/tor")
	if err != nil {
		return "", err
	}
	log.Printf("Wrote %d bytes to %s", n, memfd)

	torBinMemFD = memfd

	return memfd, nil
}

func WriteToFile(path, name string) (int, error) {
	file, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		return 0, err
	}

	torBinary, err := Asset(name)
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
