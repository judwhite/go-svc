package main

import (
	"log"
	"sync"
	"time"
)

type server struct {
	data chan int

	exit chan struct{}
	wg   sync.WaitGroup
}

func (s *server) start() {
	s.data = make(chan int)
	s.exit = make(chan struct{})

	s.wg.Add(2)
	go s.startSender()
	go s.startReceiver()
}

func (s *server) stop() error {
	close(s.exit)
	s.wg.Wait()
	return nil
}

func (s *server) startSender() {
	ticker := time.NewTicker(time.Second)
	count := 1
	for {
		select {
		case <-ticker.C:
			s.data <- count
			count++
		case <-s.exit:
			s.wg.Done()
			return
		}
	}
}

func (s *server) startReceiver() {
	for {
		select {
		case n := <-s.data:
			log.Printf("%d\n", n)
		case <-s.exit:
			s.wg.Done()
			return
		}
	}
}
