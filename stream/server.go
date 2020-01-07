package stream

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"xuxinhao.com/pbsocket/mlog"
)

type Server struct {
	l     sync.RWMutex
	wg    sync.WaitGroup
	conns map[*Conn]struct{}
	stop  sync.Once
	state int32
	cb    OnPackReceived
	pr    Protocol
	stat  Statistics
}

func (s *Server) GetCurrentStat() Statistics {
	s.l.RLock()
	t := s.stat
	for k := range s.conns {
		t = AddStat(t, k.GetCurrentStat())
	}
	s.l.RUnlock()
	return t
}

func (s *Server) handleConn(conn net.Conn) {
	myc := NewConn(conn, s.cb, s.pr)
	mlog.L.Infof("accept conn: %p", myc)
	s.l.Lock()
	s.conns[myc] = struct{}{}
	s.l.Unlock()
	s.wg.Add(1)
	defer func() {
		s.l.Lock()
		delete(s.conns, myc)
		s.stat = AddStat(s.stat, myc.stat)
		s.l.Unlock()
		s.wg.Done()
		mlog.L.Infof("exit conn: %p, stat: %+v", myc, myc.stat)
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
			mlog.L.Debugf("accept error: %v", err)
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
		mlog.L.Infof("GracefulStop start")
		atomic.StoreInt32(&s.state, stopped)
		s.l.Lock()
		for k := range s.conns {
			k.Stop()
		}
		s.l.Unlock()
		s.wg.Wait()
		mlog.L.Infof("GracefulStop end, %+v", s.stat)
	})
}

func NewServer(cb OnPackReceived, pr Protocol) *Server {
	return &Server{
		conns: make(map[*Conn]struct{}),
		cb:    cb,
		pr:    pr,
	}
}
