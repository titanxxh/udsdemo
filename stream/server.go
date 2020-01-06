package stream

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	l     sync.RWMutex
	wg    sync.WaitGroup
	conns map[*Conn]struct{}
	stop  sync.Once
	state int32
	cb    OnPackReceived
}

func (s *Server) handleConn(conn net.Conn) {
	myc := NewConn(conn, s.cb)
	log.Printf("accept conn: %p", myc)
	s.l.Lock()
	s.conns[myc] = struct{}{}
	s.l.Unlock()
	s.wg.Add(1)
	defer func() {
		s.l.Lock()
		delete(s.conns, myc)
		s.l.Unlock()
		s.wg.Done()
		log.Printf("exit conn: %p", myc)
	}()
	myc.RecvLoop()
}

func (s *Server) Serve(l net.Listener) {
	defer s.wg.Done()
	s.wg.Add(1)
	for {
		if !s.isRunning() {
			return
		}
		fd, err := l.Accept()
		if err != nil {
			// todo if tcp listener, copy the implementation from grpc/server.go
			// detect network Temporary() error, backoff accept time.
			time.Sleep(time.Millisecond * 100)
			log.Printf("accept error: %v", err)
			continue
		}
		go s.handleConn(fd)
	}
}

func (s *Server) isRunning() bool {
	return atomic.LoadInt32(&s.state) == normal
}

func (s *Server) GracefulStop() {
	s.stop.Do(func() {
		log.Printf("GracefulStop start")
		atomic.StoreInt32(&s.state, stopped)
		s.l.Lock()
		for k := range s.conns {
			k.Stop()
		}
		s.l.Unlock()
		s.wg.Wait()
		log.Printf("GracefulStop end")
	})
}

func NewServer(cb OnPackReceived) *Server {
	return &Server{
		conns: make(map[*Conn]struct{}),
		cb:    cb,
	}
}
