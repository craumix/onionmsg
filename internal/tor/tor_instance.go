package tor

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/craumix/onionmsg/pkg/types"
	"github.com/wybiral/torgo"
	"golang.org/x/net/proxy"
)

var (
	DefaultConf = Conf{
		SocksPort:   9050,
		ControlPort: 9051,
		DataDir:     "./tordir",
		TorRC:       "./torrc",
	}
)

//Instance represents an instance of a Tor process.
//It can be stopped using the Stop() CancelFunc.
type Instance struct {
	Proxy  proxy.Dialer
	Config Conf
	Stop   context.CancelFunc

	controller *torgo.Controller
	process    *os.Process
	logBuffer  *bytes.Buffer
	ctx        context.Context
	controlPW  string
	binaryPath string
}

//Conf is used to pass config values to create a Tor Instance.
type Conf struct {
	SocksPort   int
	ControlPort int
	DataDir     string
	TorRC       string
}

//NewInstance creates a Instance of a running to process.
func NewInstance(ctx context.Context, conf Conf) (*Instance, error) {
	torBinary, err := torBinaryPath()
	if err != nil {
		return nil, err
	}

	err = checkTorV3Support(torBinary)
	if err != nil {
		return nil, err
	}

	absPath, _ := exec.LookPath(torBinary)
	absPath, _ = filepath.Abs(absPath)
	instance := &Instance{
		Config:     conf,
		binaryPath: absPath,
	}
	instance.ctx, instance.Stop = context.WithCancel(ctx)

	instance.controlPW = types.RandomString(64)
	err = instance.runBinary()
	if err != nil {
		return nil, err
	}

	log.Printf("Tor seems to be runnning, pid: %d\n", instance.process.Pid)

	instance.controller, err = instance.connectController(instance.ctx)
	if err != nil {
		return nil, err
	}

	v, _ := instance.controller.GetVersion()
	log.Printf("Connected controller to tor version %s\n", v)

	instance.Proxy, _ = proxy.SOCKS5("tcp", "127.0.0.1:"+strconv.Itoa(conf.SocksPort), nil, nil)

	go func() {
		<-instance.ctx.Done()
		os.RemoveAll(instance.Config.DataDir)
	}()

	return instance, nil
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

	i.process, i.logBuffer, err = runExecutable(i.ctx, i.binaryPath, args, true)
	return err
}

//RegisterService registers a new V3 Hidden Service, and proxies the requests to the specified local port.
func (i *Instance) RegisterService(key ed25519.PrivateKey, torPort, localPort int) error {
	s, err := torgo.OnionFromEd25519(key)
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
func (i *Instance) DeregisterService(key ed25519.PrivateKey) error {
	sid, err := torgo.ServiceIDFromEd25519(ed25519.PublicKey(key[32:]))
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
		if err == nil || timeoutCtx.Err() == context.Canceled {
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
