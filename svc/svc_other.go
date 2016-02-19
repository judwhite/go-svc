// +build !windows

package svc

import "os"

// Run runs your Service.
func Run(service Service) error {
	env := environment{}
	if err := service.Init(env); err != nil {
		return err
	}

	if err := service.Start(); err != nil {
		return err
	}

	sigChan := make(chan os.Signal)
	signalNotify(sigChan, os.Interrupt, os.Kill)
	<-sigChan

	return service.Stop()
}

type environment struct{}

func (environment) IsWindowsService() bool {
	return false
}
