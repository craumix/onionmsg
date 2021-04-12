package tor

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/Craumix/tormsg/internal/bindata"
	"github.com/Craumix/tormsg/internal/memfd"
)

func Run(useInternal bool) error {
	var err error
	exe := "tor"

	if useInternal {
		exe, err = binToMem()
		if err != nil {
			return err
		}
	}

	err = runExecutable(exe)
	if err != nil {
		return err
	}

	return nil
}

func binToMem() (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("Can only execute in-memory on linux")
	}

	memfd, err := memfd.CreateMemFD("tormemfd")
	if err != nil {
		return "", err
	}

	n, err := bindata.WriteToFile(memfd, "build/tor/tor")
	if err != nil {
		return "", err
	}
	log.Printf("Wrote %d bytes to %s", n, memfd)

	return memfd, nil
}

func runExecutable(exe string) error {
	version, err := versionFromExe(exe)
	if err != nil {
		return err
	}
	log.Printf("Detected %s\n", version)

	cmd := exec.Command(exe)
	cmd.Env = os.Environ()
	cmd.Args = append([]string{"procname"}, []string{}...)

	logfile, err := os.OpenFile("tor.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	logWriter := io.Writer(logfile)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	log.Println("Starting Tor...")
	err = cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func versionFromExe(exe string) (string, error) {
	out, err := exec.Command(exe, "--version").Output()
	if err != nil {
		return "", err
	}

	version := string(out)

	return version[:len(version) - 1], nil
}