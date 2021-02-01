# go-svc

[![Go Reference](https://pkg.go.dev/badge/github.com/judwhite/go-svc.svg)](https://pkg.go.dev/github.com/judwhite/go-svc)
[![MIT License](https://img.shields.io/badge/license-MIT-007d9c)](https://github.com/judwhite/go-svc/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/judwhite/go-svc)](https://goreportcard.com/report/github.com/judwhite/go-svc)
[![Build Status](https://github.com/judwhite/go-svc/workflows/tests/badge.svg)](https://github.com/judwhite/go-svc/actions?query=workflow%3Atests)

Go Windows Service wrapper that plays nice with Linux. Windows tests [here](https://github.com/judwhite/go-svc/blob/main/svc_windows_test.go).

## Project Status

- Used in Production.
- Maintained. Issues and Pull Requests will be responded to.

## Go Modules

* Please note the `import` path and `go.mod` change from `github.com/judwhite/go-svc/svc` to `github.com/judwhite/go-svc` for `v1.2+`
* `v1.1.3` and earlier can be imported using the previous import path
* `v1.2+` code is backwards compatible with previous versions

```nginx no-this-isnt-nginx-but-the-syntax-highlighting-works
module awesomeProject

go 1.15

require github.com/judwhite/go-svc v1.2.0
```

```go
import "github.com/judwhite/go-svc"
```

## Example

```go
package main

import (
	"log"
	"sync"

	"github.com/judwhite/go-svc"
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

See the [example](https://github.com/judwhite/go-svc/tree/main/example) directory for more examples, including installing and uninstalling binaries built in Go as Windows services.

## Similar Projects

- https://github.com/kardianos/service

## License

`go-svc` is under the MIT license. See the [LICENSE](https://github.com/judwhite/go-svc/blob/main/LICENSE) file for details.
