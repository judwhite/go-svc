// +build !windows

package svc

import (
	"os"
	"syscall"
)

// Run runs your Service.
//
// Run will block until one of the signals specified in sig is received or a provided context is done.
// If sig is empty syscall.SIGINT and syscall.SIGTERM are used by default.
func Run(service Service, sig ...os.Signal) error {
	env := environment{}
	if err := service.Init(env); err != nil {
		return err
	}

	if err := service.Start(); err != nil {
		return err
	}

	if len(sig) == 0 {
		sig = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	signalChan := make(chan os.Signal, 1)
	signalNotify(signalChan, sig...)

	var doneChan <-chan struct{}
	if s, ok := service.(Context); ok {
		doneChan = s.Context().Done()
	}

	select {
	case <-signalChan:
	case <-doneChan:
	}

	return service.Stop()
}

type environment struct{}

func (environment) IsWindowsService() bool {
	return false
}
