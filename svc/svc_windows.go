// +build windows

package svc

import (
	"os"
	"sync"

	wsvc "golang.org/x/sys/windows/svc"
)

// Create variables for svc and signal functions so we can mock them in tests
var svcIsAnInteractiveSession = wsvc.IsAnInteractiveSession
var svcRun = wsvc.Run

type windowsService struct {
	i             Service
	errSync       sync.Mutex
	stopStartErr  error
	isInteractive bool
	Name          string
}

// Run runs your Service.
func Run(service Service) error {
	var err error

	interactive, err := svcIsAnInteractiveSession()
	if err != nil {
		return err
	}

	ws := &windowsService{
		i:             service,
		isInteractive: interactive,
	}

	if err = service.Init(ws); err != nil {
		return err
	}

	return ws.run()
}

func (ws *windowsService) setError(err error) {
	ws.errSync.Lock()
	ws.stopStartErr = err
	ws.errSync.Unlock()
}

func (ws *windowsService) getError() error {
	ws.errSync.Lock()
	err := ws.stopStartErr
	ws.errSync.Unlock()
	return err
}

func (ws *windowsService) IsWindowsService() bool {
	return !ws.isInteractive
}

func (ws *windowsService) run() error {
	ws.setError(nil)
	if ws.IsWindowsService() {
		// Return error messages from start and stop routines
		// that get executed in the Execute method.
		// Guarded with a mutex as it may run a different thread
		// (callback from windows).
		runErr := svcRun(ws.Name, ws)
		startStopErr := ws.getError()
		if startStopErr != nil {
			return startStopErr
		}
		if runErr != nil {
			return runErr
		}
		return nil
	}
	err := ws.i.Start()
	if err != nil {
		return err
	}

	sigChan := make(chan os.Signal)

	signalNotify(sigChan, os.Interrupt, os.Kill)

	<-sigChan

	err = ws.i.Stop()

	return err
}

// Execute is invoked by Windows
func (ws *windowsService) Execute(args []string, r <-chan wsvc.ChangeRequest, changes chan<- wsvc.Status) (bool, uint32) {
	const cmdsAccepted = wsvc.AcceptStop | wsvc.AcceptShutdown
	changes <- wsvc.Status{State: wsvc.StartPending}

	if err := ws.i.Start(); err != nil {
		ws.setError(err)
		return true, 1
	}

	changes <- wsvc.Status{State: wsvc.Running, Accepts: cmdsAccepted}
loop:
	for {
		c := <-r
		switch c.Cmd {
		case wsvc.Interrogate:
			changes <- c.CurrentStatus
		case wsvc.Stop, wsvc.Shutdown:
			changes <- wsvc.Status{State: wsvc.StopPending}
			err := ws.i.Stop()
			if err != nil {
				ws.setError(err)
				return true, 2
			}
			break loop
		default:
			continue loop
		}
	}

	return false, 0
}
