// +build !windows

package svc

import (
	"os"
	"syscall"
	"testing"

	"github.com/judwhite/go-svc/svc/internal/test"
)

func TestSignalNotify(t *testing.T) {
	for _, osSignal := range []os.Signal{syscall.SIGINT, syscall.SIGTERM} {
		testSignalNotify(t, osSignal)
	}
}

func testSignalNotify(t *testing.T, osSignal os.Signal) {
	// arrange

	// sigChan is the chan we'll send to here. if a signal matches a registered signal
	// type in the Run function (in svc_other.go) the signal will be delegated to the
	// channel passed to signalNotify, which is created in the Run function in svc_other.go.
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
		sigChan <- osSignal
	}()

	// act
	if err := Run(prg); err != nil {
		t.Fatal(err)
	}

	// assert
	test.Equal(t, 1, startCalled)
	test.Equal(t, 1, stopCalled)
	test.Equal(t, 1, initCalled)
}
