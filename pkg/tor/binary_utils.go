package tor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func runExecutable(ctx context.Context, binaryPath string, args []string, inheritEnv bool, stdout, stderr io.Writer) (*os.Process, *bytes.Buffer, error) {
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	if inheritEnv {
		cmd.Env = os.Environ()
	}

	logBuffer := new(bytes.Buffer)
	if stdout != nil {
		cmd.Stdout = io.MultiWriter(logBuffer, stdout)
	} else {
		cmd.Stdout = logBuffer
	}
	if stderr != nil {
		cmd.Stderr = io.MultiWriter(logBuffer, stderr)
	} else {
		cmd.Stderr = logBuffer
	}

	err := cmd.Start()
	if err != nil {
		return nil, nil, err
	}

	return cmd.Process, logBuffer, nil
}

func checkTorV3Support(binaryPath string) error {
	minTorVersion := "0.3.3.6"

	version, err := versionFromBinary(binaryPath)
	if err != nil {
		return fmt.Errorf("unable to determine Tor Version: \"%s\"", err.Error())
	}
	if version < minTorVersion {
		return fmt.Errorf("tor version to old %s, at least %s is required", version, minTorVersion)
	}

	return nil
}

func versionFromBinary(binaryPath string) (string, error) {
	raw, err := getExecutionOutput(binaryPath, "--version")
	if err != nil {
		return "", err
	}

	if strings.Contains(raw, "\n") {
		raw = raw[:strings.Index(raw, "\n")]
	}

	return strings.Split(raw[12:len(raw)-1], " ")[0], nil
}

func pwHashFromBinary(binaryPath, pw string) (string, error) {
	return getExecutionOutput(binaryPath, "--hash-password", pw)
}

func getExecutionOutput(binaryPath string, args ...string) (string, error) {
	r, err := exec.Command(binaryPath, args...).Output()
	if err != nil {
		return "", err
	}

	return strings.Trim(string(r), "\n"), nil
}
