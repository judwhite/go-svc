// +build windows

package svc

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	wsvc "golang.org/x/sys/windows/svc"
)

// Create variables for svc and signal functions so we can mock them in tests
var svcIsWindowsService = wsvc.IsWindowsService
var svcRun = wsvc.Run

type windowsService struct {
	i                Service
	errSync          sync.Mutex
	stopStartErr     error
	isWindowsService bool
	signals          []os.Signal
	Name             string
	ctx              context.Context
}

// Run runs an implementation of the Service interface.
//
// Run will block until the Windows Service is stopped or Ctrl+C is pressed if
// running from the console.
//
// Stopping the Windows Service and Ctrl+C will call the Service's Stop method to
// initiate a graceful shutdown.
//
// Note that WM_CLOSE is not handled (end task) and the Service's Stop method will
// not be called.
//
// The sig parameter is to keep parity with the non-Windows API. Only syscall.SIGINT
// (Ctrl+C) can be handled on Windows. Nevertheless, you can override the default
// signals which are handled by specifying sig.
func Run(service Service, sig ...os.Signal) error {
	var err error

	isWindowsService, err := svcIsWindowsService()
	if err != nil {
		return err
	}

	if len(sig) == 0 {
		sig = []os.Signal{syscall.SIGINT}
	}

	var ctx context.Context
	if s, ok := service.(Context); ok {
		ctx = s.Context()
	} else {
		ctx = context.Background()
	}

	ws := &windowsService{
		i:                service,
		isWindowsService: isWindowsService,
		signals:          sig,
		ctx:              ctx,
	}

	if ws.IsWindowsService() {
		// the working directory for a Windows Service is C:\Windows\System32
		// this is almost certainly not what the user wants.
		dir := filepath.Dir(os.Args[0])
		if err = os.Chdir(dir); err != nil {
			return err
		}
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
	return ws.isWindowsService
}

func (ws *windowsService) run() error {
	ws.setError(nil)
	if ws.IsWindowsService() {
		// Return error messages from start and stop routines
		// that get executed in the Execute method.
		// Guarded with a mutex as it may run a different thread
		// (callback from Windows).
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

	signalChan := make(chan os.Signal, 1)
	signalNotify(signalChan, ws.signals...)

	select {
	case <-signalChan:
	case <-ws.ctx.Done():
	}

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

	for {
		var c wsvc.ChangeRequest
		select {
		case c = <-r:
		case <-ws.ctx.Done():
			c = wsvc.ChangeRequest{Cmd: wsvc.Stop}
		}

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
			return false, 0
		default:
		}
	}
}
