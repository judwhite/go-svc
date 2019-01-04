package svc

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

type mockProgram struct {
	start func() error
	stop  func() error
	init  func(Environment) error
}

func (p *mockProgram) Start() error {
	return p.start()
}

func (p *mockProgram) Stop() error {
	return p.stop()
}

func (p *mockProgram) Init(wse Environment) error {
	return p.init(wse)
}

func makeProgram(startCalled, stopCalled, initCalled *int) *mockProgram {
	return &mockProgram{
		start: func() error {
			*startCalled++
			return nil
		},
		stop: func() error {
			*stopCalled++
			return nil
		},
		init: func(wse Environment) error {
			*initCalled++
			return nil
		},
	}
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
