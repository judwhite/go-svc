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
	ticker := time.NewTicker(20 * time.Millisecond)
	defer s.wg.Done()
	count := 1
	for {
		select {
		case <-ticker.C:
			select {
			case s.data <- count:
				count++
			case <-s.exit:
				// if the other goroutine exits there'll be no one to receive on the data chan,
				// and this goroutine could block. you can simulate this by putting a time.Sleep
				// inside startReceiver's receive on s.data and a log.Println here
				return
			}
		case <-s.exit:
			return
		}
	}
}

func (s *server) startReceiver() {
	defer s.wg.Done()
	for {
		select {
		case n := <-s.data:
			log.Printf("%d\n", n)
		case <-s.exit:
			return
		}
	}
}
