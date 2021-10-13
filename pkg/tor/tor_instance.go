package tor

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/wybiral/torgo"
	"golang.org/x/net/proxy"
)

//Instance represents an instance of a Tor process.
//It can be stopped using the Stop() CancelFunc.
type Instance struct {
	Proxy  proxy.Dialer
	Config Config
	Stop   context.CancelFunc

	controller *torgo.Controller
	process    *os.Process
	logBuffer  *bytes.Buffer
	ctx        context.Context
	controlPW  string
	binaryPath string
}

//Config is used to pass config values to create a Tor Instance.
type Config struct {
	SocksPort, ControlPort int
	DataDir, Binary, TorRC string
	ControlPass            bool
	StdOut, StdErr         io.Writer
}

func DefaultConfig() Config {
	return Config{
		SocksPort:   9050,
		ControlPort: 9051,
		DataDir:     "tor",
		TorRC:       "torrc",
		ControlPass: true,
		Binary:      "tor",
	}
}

//NewInstance creates an Instance of a running to process.
func NewInstance(conf Config) (*Instance, error) {
	var err error
	torBinary := conf.Binary

	if torBinary == "" {
		torBinary, err = torBinaryPath()
		if err != nil {
			return nil, err
		}
	}

	absPath, _ := exec.LookPath(torBinary)
	absPath, _ = filepath.Abs(absPath)
	instance := &Instance{
		Config:     conf,
		binaryPath: absPath,
	}

	if conf.ControlPass {
		instance.controlPW = prngString(64)
	}

	return instance, nil
}
func (i *Instance) Start(ctx context.Context) error {
	i.ctx, i.Stop = context.WithCancel(ctx)

	err := checkTorV3Support(i.binaryPath)
	if err != nil {
		return err
	}

	err = i.runBinary()
	if err != nil {
		return err
	}

	i.controller, err = i.connectController(i.ctx)
	if err != nil {
		return err
	}

	i.Proxy, err = proxy.SOCKS5("tcp", "127.0.0.1:"+strconv.Itoa(i.Config.SocksPort), nil, nil)
	if err != nil {
		return err
	}

	return nil
}

func (i *Instance) runBinary() error {
	err := os.MkdirAll(i.Config.DataDir, 0700)
	if err != nil {
		return err
	}

	args := []string{"-f", i.Config.TorRC, "--ignore-missing-torrc",
		"SocksPort", strconv.Itoa(i.Config.SocksPort),
		"ControlPort", strconv.Itoa(i.Config.ControlPort),
		"DataDirectory", i.Config.DataDir}

	if i.controlPW != "" {
		hash, err := pwHashFromBinary(i.binaryPath, i.controlPW)
		if err != nil {
			return err
		}

		args = append(args, "HashedControlPassword", hash)
	}

	i.process, i.logBuffer, err = runExecutable(i.ctx, i.binaryPath, args, true, i.Config.StdOut, i.Config.StdErr)
	return err
}

//RegisterService registers a new V3 Hidden Service, and proxies the requests to the specified local port.
func (i *Instance) RegisterService(priv ed25519.PrivateKey, torPort, localPort int) error {
	s, err := torgo.OnionFromEd25519(priv)
	if err != nil {
		return err
	}

	s.Ports[torPort] = "127.0.0.1:" + strconv.Itoa(localPort)

	err = i.controller.AddOnion(s)
	if err != nil {
		return err
	}

	return nil
}

//DeregisterService removes a HiddenService.
func (i *Instance) DeregisterService(pub ed25519.PublicKey) error {
	sid, err := torgo.ServiceIDFromEd25519(pub)
	if err != nil {
		return err
	}

	err = i.controller.DeleteOnion(sid)
	if err != nil {
		return err
	}

	return nil
}

func (i *Instance) connectController(ctx context.Context) (*torgo.Controller, error) {
	var (
		err  error
		ctrl *torgo.Controller
	)

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	for {
		ctrl, err = torgo.NewController("127.0.0.1:" + strconv.Itoa(i.Config.ControlPort))
		if err == nil || timeoutCtx.Err() != nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	if i.controlPW == "" {
		err = ctrl.AuthenticateNone()
	} else {
		err = ctrl.AuthenticatePassword(i.controlPW)
	}
	if err != nil {
		return nil, err
	}

	return ctrl, nil
}

//Log returns the output of STDOUT and STDERR from the Tor process.
func (i *Instance) Log() string {
	return i.logBuffer.String()
}

func (i *Instance) Pid() int {
	return i.process.Pid
}

func (i *Instance) Version() string {
	v, err := i.controller.GetVersion()
	if err != nil {
		return "error"
	}
	return v
}

func (i *Instance) BinaryPath() string {
	return i.binaryPath
}

// Info returns the log of the used to instance.
func (i *Instance) Info() interface{} {
	return struct {
		Log        string `json:"log"`
		Version    string `json:"version"`
		PID        int    `json:"pid"`
		BinaryPath string `json:"path"`
	}{
		i.Log(),
		i.Version(),
		i.Pid(),
		i.BinaryPath(),
	}
}
