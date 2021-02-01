// +build windows

package svc

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"syscall"
	"testing"
	"time"

	wsvc "golang.org/x/sys/windows/svc"
)

func setupWinServiceTest(wsf *mockWinServiceFuncs) {
	// wsfWrapper allows signalNotify, svcIsWindowsService, and svcRun to be set once.
	// Individual test functions set "wsf" to add behavior.
	wsfWrapper := &mockWinServiceFuncs{
		signalNotify: func(c chan<- os.Signal, sig ...os.Signal) {
			if c == nil {
				panic("os/signal: Notify using nil channel")
			}

			if wsf.signalNotify != nil {
				wsf.signalNotify(c, sig...)
			} else {
				wsf1 := *wsf
				go func() {
					for val := range wsf1.sigChan {
						for _, registeredSig := range sig {
							if val == registeredSig {
								c <- val
							}
						}
					}
				}()
			}
		},
		svcIsWindowsService: func() (bool, error) {
			return wsf.svcIsWindowsService()
		},
		svcRun: func(name string, handler wsvc.Handler) error {
			return wsf.svcRun(name, handler)
		},
	}

	signalNotify = wsfWrapper.signalNotify
	svcIsWindowsService = wsfWrapper.svcIsWindowsService
	svcRun = wsfWrapper.svcRun
}

type mockWinServiceFuncs struct {
	signalNotify          func(chan<- os.Signal, ...os.Signal)
	svcIsWindowsService   func() (bool, error)
	sigChan               chan os.Signal
	svcRun                func(string, wsvc.Handler) error
	ws                    *windowsService
	executeReturnedBool   bool
	executeReturnedUInt32 uint32
	changes               []wsvc.Status
}

func setWindowsServiceFuncs(isWindowsService bool, onRunningSendCmd *wsvc.Cmd) (*mockWinServiceFuncs, chan<- wsvc.ChangeRequest) {
	changeRequestChan := make(chan wsvc.ChangeRequest, 4)
	changesChan := make(chan wsvc.Status)
	done := make(chan struct{})

	var wsf *mockWinServiceFuncs
	wsf = &mockWinServiceFuncs{
		sigChan: make(chan os.Signal),
		svcIsWindowsService: func() (bool, error) {
			return isWindowsService, nil
		},
		svcRun: func(name string, handler wsvc.Handler) error {
			ws, ok := handler.(*windowsService)
			if !ok {
				return errors.New("handler is not a windowsService")
			}
			wsf.ws = ws
			wsf.executeReturnedBool, wsf.executeReturnedUInt32 = handler.Execute(nil, changeRequestChan, changesChan)
			done <- struct{}{}
			return nil
		},
	}

	var currentState wsvc.State

	go func() {
	loop:
		for {
			select {
			case change := <-changesChan:
				wsf.changes = append(wsf.changes, change)
				currentState = change.State

				if change.State == wsvc.Running && onRunningSendCmd != nil {
					changeRequestChan <- wsvc.ChangeRequest{
						Cmd:           *onRunningSendCmd,
						CurrentStatus: wsvc.Status{State: currentState},
					}
				}
			case <-done:
				break loop
			}
		}
	}()

	setupWinServiceTest(wsf)

	return wsf, changeRequestChan
}

func TestWinService_RunWindowsService_NonInteractive(t *testing.T) {
	for _, svcCmd := range []wsvc.Cmd{wsvc.Stop, wsvc.Shutdown} {
		testRunWindowsServiceNonInteractive(t, svcCmd)
	}
}

func testRunWindowsServiceNonInteractive(t *testing.T, svcCmd wsvc.Cmd) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)

	wsf, _ := setWindowsServiceFuncs(true, &svcCmd)

	// act
	if err := Run(prg); err != nil {
		t.Fatal(err)
	}

	// assert
	changes := wsf.changes

	equal(t, 1, startCalled)
	equal(t, 1, stopCalled)
	equal(t, 1, initCalled)

	equal(t, 3, len(changes))
	equal(t, wsvc.StartPending, changes[0].State)
	equal(t, wsvc.Running, changes[1].State)
	equal(t, wsvc.StopPending, changes[2].State)

	equal(t, false, wsf.executeReturnedBool)
	equal(t, uint32(0), wsf.executeReturnedUInt32)

	assertNil(t, wsf.ws.getError())
}

func TestRunWindowsServiceNonInteractive_StartError(t *testing.T) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)
	prg.start = func() error {
		startCalled++
		return errors.New("start error")
	}

	svcStop := wsvc.Stop
	wsf, _ := setWindowsServiceFuncs(true, &svcStop)

	// act
	err := Run(prg)

	// assert
	equal(t, "start error", err.Error())

	changes := wsf.changes

	equal(t, 1, startCalled)
	equal(t, 0, stopCalled)
	equal(t, 1, initCalled)

	equal(t, 1, len(changes))
	equal(t, wsvc.StartPending, changes[0].State)

	equal(t, true, wsf.executeReturnedBool)
	equal(t, uint32(1), wsf.executeReturnedUInt32)

	equal(t, "start error", wsf.ws.getError().Error())
}

func TestRunWindowsServiceInteractive_StartError(t *testing.T) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)
	prg.start = func() error {
		startCalled++
		return errors.New("start error")
	}

	wsf, _ := setWindowsServiceFuncs(false, nil)

	// act
	err := Run(prg)

	// assert
	equal(t, "start error", err.Error())

	changes := wsf.changes

	equal(t, 1, startCalled)
	equal(t, 0, stopCalled)
	equal(t, 1, initCalled)

	equal(t, 0, len(changes))
}

func TestRunWindowsService_BeforeStartError(t *testing.T) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)
	prg.init = func(Environment) error {
		initCalled++
		return errors.New("before start error")
	}

	wsf, _ := setWindowsServiceFuncs(true, nil)

	// act
	err := Run(prg)

	// assert
	equal(t, "before start error", err.Error())

	changes := wsf.changes

	equal(t, 0, startCalled)
	equal(t, 0, stopCalled)
	equal(t, 1, initCalled)

	equal(t, 0, len(changes))
}

func TestRunWindowsService_IsWindowsServiceError(t *testing.T) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)

	wsf, _ := setWindowsServiceFuncs(true, nil)
	wsf.svcIsWindowsService = func() (bool, error) {
		return false, errors.New("IsWindowsService error")
	}

	// act
	err := Run(prg)

	// assert
	equal(t, "IsWindowsService error", err.Error())

	changes := wsf.changes

	equal(t, 0, startCalled)
	equal(t, 0, stopCalled)
	equal(t, 0, initCalled)

	equal(t, 0, len(changes))
}

func TestRunWindowsServiceNonInteractive_RunError(t *testing.T) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)

	svcStop := wsvc.Stop
	wsf, _ := setWindowsServiceFuncs(true, &svcStop)
	wsf.svcRun = func(name string, handler wsvc.Handler) error {
		ws, ok := handler.(*windowsService)
		if !ok {
			return errors.New("handler is not a windowsService")
		}
		wsf.ws = ws
		return errors.New("wsvc.Run error")
	}

	// act
	err := Run(prg)

	// assert
	equal(t, "wsvc.Run error", err.Error())

	changes := wsf.changes

	equal(t, 0, startCalled)
	equal(t, 0, stopCalled)
	equal(t, 1, initCalled)

	equal(t, 0, len(changes))

	assertNil(t, wsf.ws.getError())
}

func TestRunWindowsServiceNonInteractive_Interrogate(t *testing.T) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)

	wsf, changeRequest := setWindowsServiceFuncs(true, nil)

	time.AfterFunc(50*time.Millisecond, func() {
		// ignored, PausePending won't be in changes slice
		// make sure we don't panic/err on unexpected values
		changeRequest <- wsvc.ChangeRequest{
			Cmd:           wsvc.Pause,
			CurrentStatus: wsvc.Status{State: wsvc.PausePending},
		}
	})

	time.AfterFunc(100*time.Millisecond, func() {
		// handled, Paused will be in changes slice
		changeRequest <- wsvc.ChangeRequest{
			Cmd:           wsvc.Interrogate,
			CurrentStatus: wsvc.Status{State: wsvc.Paused},
		}
	})

	time.AfterFunc(200*time.Millisecond, func() {
		// handled, but CurrentStatus overridden with StopPending;
		// ContinuePending won't be in changes slice
		changeRequest <- wsvc.ChangeRequest{
			Cmd:           wsvc.Stop,
			CurrentStatus: wsvc.Status{State: wsvc.ContinuePending},
		}
	})

	// act
	if err := Run(prg); err != nil {
		t.Fatal(err)
	}

	// assert
	changes := wsf.changes

	equal(t, 1, startCalled)
	equal(t, 1, stopCalled)
	equal(t, 1, initCalled)

	equal(t, 4, len(changes))
	equal(t, wsvc.StartPending, changes[0].State)
	equal(t, wsvc.Running, changes[1].State)
	equal(t, wsvc.Paused, changes[2].State)
	equal(t, wsvc.StopPending, changes[3].State)

	equal(t, false, wsf.executeReturnedBool)
	equal(t, uint32(0), wsf.executeReturnedUInt32)

	assertNil(t, wsf.ws.getError())
}

func TestRunWindowsServiceInteractive_StopError(t *testing.T) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)
	prg.stop = func() error {
		stopCalled++
		return errors.New("stop error")
	}

	wsf, _ := setWindowsServiceFuncs(false, nil)

	go func() {
		wsf.sigChan <- os.Interrupt
	}()

	// act
	err := Run(prg)

	// assert
	equal(t, "stop error", err.Error())
	equal(t, 1, startCalled)
	equal(t, 1, stopCalled)
	equal(t, 1, initCalled)
	equal(t, 0, len(wsf.changes))
}

func TestRunWindowsServiceNonInteractive_StopError(t *testing.T) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)
	prg.stop = func() error {
		stopCalled++
		return errors.New("stop error")
	}

	shutdownCmd := wsvc.Shutdown
	wsf, _ := setWindowsServiceFuncs(true, &shutdownCmd)

	// act
	err := Run(prg)

	// assert
	changes := wsf.changes

	equal(t, "stop error", err.Error())

	equal(t, 1, startCalled)
	equal(t, 1, stopCalled)
	equal(t, 1, initCalled)

	equal(t, 3, len(changes))
	equal(t, wsvc.StartPending, changes[0].State)
	equal(t, wsvc.Running, changes[1].State)
	equal(t, wsvc.StopPending, changes[2].State)

	equal(t, true, wsf.executeReturnedBool)
	equal(t, uint32(2), wsf.executeReturnedUInt32)

	equal(t, "stop error", wsf.ws.getError().Error())
}

func TestDefaultSignalHandling(t *testing.T) {
	signals := []os.Signal{syscall.SIGINT} // default signal handled
	for _, signal := range signals {
		testSignalNotify(t, signal)
	}
}

func TestUserDefinedSignalHandling(t *testing.T) {
	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}
	for _, signal := range signals {
		testSignalNotify(t, signal, signals...)
	}
}

func testSignalNotify(t *testing.T, signal os.Signal, sig ...os.Signal) {
	// arrange
	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)

	wsf, _ := setWindowsServiceFuncs(false, nil)

	go func() {
		wsf.sigChan <- signal
	}()

	// act
	if err := Run(prg, sig...); err != nil {
		t.Fatal(err)
	}

	// assert
	equal(t, 1, startCalled)
	equal(t, 1, stopCalled)
	equal(t, 1, initCalled)
	equal(t, 0, len(wsf.changes))
}

func equal(t *testing.T, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		_, file, line, _ := runtime.Caller(1)
		t.Logf("\033[31m%s:%d:\n\n\t   %#v (expected)\n\n\t!= %#v (actual)\033[39m\n\n",
			filepath.Base(file), line, expected, actual)
		t.FailNow()
	}
}

func assertNil(t *testing.T, object interface{}) {
	if !isNil(object) {
		_, file, line, _ := runtime.Caller(1)
		t.Logf("\033[31m%s:%d:\n\n\t   <nil> (expected)\n\n\t!= %#v (actual)\033[39m\n\n",
			filepath.Base(file), line, object)
		t.FailNow()
	}
}

func isNil(object interface{}) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
	if kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil() {
		return true
	}

	return false
}
