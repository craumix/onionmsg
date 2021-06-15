package tor

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/craumix/onionmsg/pkg/types"
	"github.com/wybiral/torgo"
	"golang.org/x/net/proxy"
)

type TorInstance struct {
	Process    *os.Process
	Controller *torgo.Controller
	Proxy      proxy.Dialer
	LogBuffer  *bytes.Buffer

	tordir      string
	socksPort   int
	controlPort int
}

func NewTorInstance(tordir string, socksPort, controlPort int) (instance *TorInstance, err error) {
	pw := types.RandomString(64)
	torproc, logBuffer, err := launchTor(pw, tordir, socksPort, controlPort)
	if err != nil {
		return
	}

	log.Printf("Tor seems to be runnning, pid: %d\n", torproc.Pid)

	controller, err := WaitForController(pw, "127.0.0.1:"+strconv.Itoa(controlPort), time.Second, 30)
	if err != nil {
		return
	}

	v, _ := controller.GetVersion()
	log.Printf("Connected controller to tor version %s\n", v)

	dialer, _ := proxy.SOCKS5("tcp", "127.0.0.1:"+strconv.Itoa(socksPort), nil, nil)

	instance = &TorInstance{
		Process:     torproc,
		Controller:  controller,
		Proxy:       dialer,
		LogBuffer:   logBuffer,
		tordir:      tordir,
		socksPort:   socksPort,
		controlPort: controlPort,
	}

	return
}

func (i *TorInstance) Stop() (err error) {
	if i.Process != nil {
		if runtime.GOOS == "windows" {
			err = i.Process.Kill()
		} else {
			err = i.Process.Signal(os.Interrupt)
		}
	} else {
		err = fmt.Errorf("tor was not running")
		return
	}

	if err != nil {
		return
	}

	//Give Tor some time to stop and drop file locks
	time.Sleep(time.Millisecond * 500)

	err = i.cleanup()

	return
}

func (i *TorInstance) cleanup() (err error) {
	err = os.RemoveAll(i.tordir)
	return
}

func (i *TorInstance) RegisterService(key ed25519.PrivateKey, torPort, localPort int) error {
	s, err := torgo.OnionFromEd25519(key)
	if err != nil {
		return err
	}

	s.Ports[torPort] = "127.0.0.1:" + strconv.Itoa(localPort)

	err = i.Controller.AddOnion(s)
	if err != nil {
		return err
	}

	return nil
}

func (i *TorInstance) DeregisterService(key ed25519.PrivateKey) error {
	sid, err := torgo.ServiceIDFromEd25519(ed25519.PublicKey(key))
	if err != nil {
		return err
	}

	err = i.Controller.DeleteOnion(sid)
	if err != nil {
		return err
	}

	return nil
}
