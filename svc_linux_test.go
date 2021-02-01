// +build !windows

package svc

import (
	"os"
	"syscall"
	"testing"
)

func TestDefaultSignalHandling(t *testing.T) {
	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM} // default signals handled
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

	// sigChan is the chan we'll send to here. if a signal matches a registered signal
	// type in the Run function (in svc_linux.go) the signal will be delegated to the
	// channel passed to signalNotify, which is created in the Run function in svc_linux.go.
	// shortly: we send here and the Run function gets it if it matches the filter.
	sigChan := make(chan os.Signal)

	var startCalled, stopCalled, initCalled int
	prg := makeProgram(&startCalled, &stopCalled, &initCalled)

	signalNotify = func(c chan<- os.Signal, sig ...os.Signal) {
		if c == nil {
			panic("os/signal: Notify using nil channel")
		}

		go func() {
			for val := range sigChan {
				for _, registeredSig := range sig {
					if val == registeredSig {
						c <- val
					}
				}
			}
		}()
	}

	go func() {
		sigChan <- signal
	}()

	// act
	if err := Run(prg, sig...); err != nil {
		t.Fatal(err)
	}

	// assert
	if startCalled != 1 {
		t.Errorf("startCalled, want: 1 got: %d", startCalled)
	}
	if stopCalled != 1 {
		t.Errorf("stopCalled, want: 1 got: %d", stopCalled)
	}
	if initCalled != 1 {
		t.Errorf("initCalled, want: 1 got: %d", initCalled)
	}
}
