# go-svc

[![Go Report Card](https://goreportcard.com/badge/github.com/judwhite/go-svc)](https://goreportcard.com/report/github.com/judwhite/go-svc)
[![Build Status](https://travis-ci.org/judwhite/go-svc.svg?branch=master)](https://travis-ci.org/judwhite/go-svc)

Go Windows Service wrapper that plays nice with Linux.

## Install

`go get -u https://github.com/judwhite/go-svc/svc`

## Example

```go
package main

import (
	"log"
	"sync"

	"github.com/judwhite/go-svc/svc"
)

// program implements svc.Service
type program struct {
	wg   sync.WaitGroup
	quit chan struct{}
}

func main() {
	prg := &program{}

	// Call svc.Run to start your program/service.
	if err := svc.Run(prg); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	log.Printf("is win service? %v\n", env.IsWindowsService())
	return nil
}

func (p *program) Start() error {
	// The Start method must not block, or Windows may assume your service failed
	// to start. Launch a Goroutine here to do something interesting/blocking.

	p.quit = make(chan struct{})

	p.wg.Add(1)
	go func() {
		log.Println("Starting...")
		<-p.quit
		log.Println("Quit signal received...")
		p.wg.Done()
	}()

	return nil
}

func (p *program) Stop() error {
	// The Stop method is invoked by stopping the Windows service, or by pressing Ctrl+C on the console.
	// This method may block, but it's a good idea to finish quickly or your process may be killed by
	// Windows during a shutdown/reboot. As a general rule you shouldn't rely on graceful shutdown.

	log.Println("Stopping...")
	close(p.quit)
	p.wg.Wait()
	log.Println("Stopped.")
	return nil
}
```

## More Examples

See [svc/example](https://github.com/judwhite/go-svc/tree/master/example) for more examples, including installing and uninstalling binaries as Windows services.

## License

go-svc is under the MIT license. See the [LICENSE](https://github.com/judwhite/go-svc/blob/master/LICENSE) file for details.
