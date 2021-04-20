package tor

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/Craumix/tormsg/internal/bindata"
	"github.com/Craumix/tormsg/internal/sio"
)

var (
	torBinMemFD	string
)

func Run(pw, datadir string, socksPort, controlPort int, useInternal bool) (*os.Process, error) {
	var err error
	exe := "tor"
	torrc := datadir + "/torrc"
	logfile := datadir + "/tor.log"

	if useInternal {
		if runtime.GOOS != "linux" {
			return nil, fmt.Errorf("Cannot use internal tor binary on platfrom \"%s\"", runtime.GOOS)
		}
		exe, err = binToMem()
		if err != nil {
			return nil, err
		}
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

func runExecutable(exe string, args []string, logpath string) (*os.Process, error) {
	version, err := versionFromExe(exe)
	if err != nil {
		return nil, err
	}
	log.Printf("Detected %s\n", version)

	cmd := exec.Command(exe)
	cmd.Env = os.Environ()
	cmd.Args = append([]string{"procname"}, args...)

	if logpath != "" {
		logfile, err := os.OpenFile(logpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Printf("Unable to open logfile \"%s\"\n%s\n", logpath, err.Error())
		}else {
			logWriter := io.Writer(logfile)
			cmd.Stdout = logWriter
			cmd.Stderr = logWriter
		}
	}

	log.Println("Starting Tor...")
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd.Process, nil
}

func binToMem() (string, error) {
	if(torBinMemFD != "") {
		return torBinMemFD, nil
	}

	memfd, err := sio.CreateMemFD("tormemfd")
	if err != nil {
		return "", err
	}

	n, err := bindata.WriteToFile(memfd, "build/tor/tor")
	if err != nil {
		return "", err
	}
	log.Printf("Wrote %d bytes to %s", n, memfd)

	torBinMemFD = memfd

	return memfd, nil
}

func versionFromExe(exe string) (string, error) {
	out, err := exec.Command(exe, "--version").Output()
	if err != nil {
		return "", err
	}

	version := string(out)

	return version[:len(version) - 1], nil
}

func pwHashFromExe(exe, pw string) (string, error) {
	out, err := exec.Command(exe, "--hash-password", pw).Output()
	if err != nil {
		return "", err
	}

	version := string(out)

	return version[:len(version) - 1], nil
}