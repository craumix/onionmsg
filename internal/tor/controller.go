package tor

import (
	"time"

	"github.com/wybiral/torgo"
)

func WaitForController(pw, addr string, interval time.Duration, retries int) (*torgo.Controller, error) {
	var err error
	var ctrl *torgo.Controller

	for(retries > 0) {
		ctrl, err = torgo.NewController(addr)
		if err == nil {
			break;
		}

		retries--;
		time.Sleep(interval)
	}
	if err != nil {
		return nil, err
	}
	
	if pw == "" {
		err = ctrl.AuthenticateNone()
	}else {
		err = ctrl.AuthenticatePassword(pw)
	}
	if err != nil {
		return nil, err
	}
	
	return ctrl, nil
}