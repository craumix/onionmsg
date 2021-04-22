// +build !internalTor

package tor

import (
	"log"
	"os"
	"strconv"
)

func Run(pw, datadir string, socksPort, controlPort int) (*os.Process, error) {
	var err error
	exe := "tor"
	torrc := datadir + "/torrc"
	logfile := datadir + "/tor.log"

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
