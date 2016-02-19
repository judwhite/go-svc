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

	signalChan := make(chan os.Signal, 1)
	signalNotify(signalChan, os.Interrupt, os.Kill)
	<-signalChan

	return service.Stop()
}

type environment struct{}

func (environment) IsWindowsService() bool {
	return false
}
