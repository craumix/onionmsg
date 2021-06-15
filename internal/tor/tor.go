package tor

//go:generate go-bindata -nometadata -nocompress -tags internalTor -o bindata.go -pkg tor ../../third_party/tor/tor

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"os/exec"
	"strconv"
)

func launchTor(pw, datadir string, socksPort, controlPort int) (*os.Process, *bytes.Buffer, error) {
	exe, err := getExePath()
	if err != nil {
		return nil, nil, err
	}

	err = os.MkdirAll(datadir, 0700)
	if err != nil {
		return nil, nil, err
	}

	torrc := "./torrc"
	args := []string{"-f", torrc,
		"--ignore-missing-torrc",
		"SocksPort", strconv.Itoa(socksPort),
		"ControlPort", strconv.Itoa(controlPort),
		"DataDirectory", datadir}

	if pw != "" {
		hash, err := pwHashFromExe(exe, pw)
		if err != nil {
			return nil, nil, err
		}

		args = append(args, "HashedControlPassword", hash)
		log.Printf("Password hash set as %s\n", hash)
	}

	logBuffer := new(bytes.Buffer)
	proc, err := runExecutable(exe, args, logBuffer)
	if err != nil {
		return nil, nil, err
	}

	return proc, logBuffer, nil
}

func runExecutable(exe string, args []string, logBuffer *bytes.Buffer) (*os.Process, error) {
	version, err := versionFromExe(exe)
	if err != nil {
		return nil, err
	}
	log.Printf("Detected %s\n", version)

	cmd := exec.Command(exe)
	cmd.Env = os.Environ()
	cmd.Args = append([]string{"procname"}, args...)

	if logBuffer != nil {
		logWriter := bufio.NewWriter(logBuffer)
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	}

	log.Println("Starting Tor...")
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd.Process, nil
}

func versionFromExe(exe string) (string, error) {
	return getExeOuput(exe, "--version")
}

func pwHashFromExe(exe, pw string) (string, error) {
	return getExeOuput(exe, "--hash-password", pw)
}

func getExeOuput(exe string, args ...string) (o string, err error) {
	var r []byte
	r, err = exec.Command(exe, args...).Output()
	if err != nil {
		return "", err
	}

	o = string(r)
	o = o[:len(o)-1]

	return o, nil
}
