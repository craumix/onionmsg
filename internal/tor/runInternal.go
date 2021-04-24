// +build internalTor

package tor

import (
	"fmt"
	"github.com/Craumix/onionmsg/internal/sio"
	"log"
	"os"
	"runtime"
	"strconv"
)

var (
	torBinMemFD string
)

func Run(pw, datadir string, socksPort, controlPort int) (*os.Process, error) {
	var err error
	var exe string

	torrc := datadir + "/torrc"
	logfile := datadir + "/tor.log"

	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("Cannot use internal tor binary on platform \"%s\"", runtime.GOOS)
	}
	exe, err = binToMem()
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(datadir, 0700)
	if err != nil {
		return nil, err
	}

	_, err = os.OpenFile(torrc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("Unable to to touch torrc \"%s\"\n%s\n", torrc, err.Error())
	}

	args := []string{"-f", torrc,
		"SocksPort", strconv.Itoa(socksPort),
		"ControlPort", strconv.Itoa(controlPort),
		"DataDirectory", datadir}

	if pw != "" {
		hash, err := pwHashFromExe(exe, pw)
		if err != nil {
			return nil, err
		}

		args = append(args, "HashedControlPassword", hash)
		log.Printf("Password hash set as %s\n", hash)
	}

	proc, err := runExecutable(exe, args, logfile)
	if err != nil {
		return nil, err
	}

	return proc, nil
}

func binToMem() (string, error) {
	if torBinMemFD != "" {
		return torBinMemFD, nil
	}

	memfd, err := sio.CreateMemFD("tormemfd")
	if err != nil {
		return "", err
	}

	n, err := WriteToFile(memfd, "build/tor/tor")
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
