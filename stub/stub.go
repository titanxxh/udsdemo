package stub

import (
	"fmt"
	"net"
	"sync"
	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/stream"
)

var (
	errPeerNotExist = fmt.Errorf("peer not exist")
)

type (
	PeerID   = uint64
	Identity = int32
	LocalCb  func(header *subpub.Header, payloadType int32, payload []byte)
)

type (
	peerInfo struct {
		id   PeerID
		gene int32
		c    *stream.Conn
	}
	Local struct {
		s        *stream.Server
		l        net.Listener
		mu       sync.RWMutex
		localCbs map[Identity]LocalCb
		peers    map[PeerID]peerInfo
		self     uint64
		selfGene int32
	}
)

const (
	Hello = iota
	HelloAck
	Goodbye
	GoodbyeAck
)

func (s *Local) hello(peer peerInfo) {
	s.mu.Lock()
	old, ok := s.peers[peer.id]
	if ok {
		if old == peer {
			s.mu.Unlock()
			return
		}
		old.c.Stop()
	}
	s.peers[peer.id] = peer
	peer.c.Send(ConstructHelloAck(s.self, s.selfGene))
	s.mu.Unlock()
}

func (s *Local) goodbye(peer peerInfo) {
	s.mu.Lock()
	p, ok := s.peers[peer.id]
	if ok {
		delete(s.peers, peer.id)
		peer.c.Send(ConstructGoodbyeAck(s.self, s.selfGene))
	}
	s.mu.Unlock()

}

func (s *Local) internal(peer peerInfo, t int32, buf []byte) {
	switch t {
	case Hello:
		s.hello(peer)
	case Goodbye:
		s.goodbye(peer)
	}
}

func (s *Local) recv(c *stream.Conn, pack stream.Packet) {
	m := pack.(*subpub.UniMessage)
	if m.Header == nil {
		return
	}
	to := m.Header.To
	if to == 0 {
		s.internal(peerInfo{id: m.Header.Id, gene: m.Header.Generation, c: c}, m.PayloadType, m.Payload)
	}
	s.mu.RLock()
	cb, ok := s.localCbs[to]
	s.mu.RUnlock()
	if !ok {
		return
	}
	cb(m.Header, m.PayloadType, m.Payload)
}

//todo add sub for Remote

func (s *Local) Send(header *subpub.Header, payloadType int32, payload []byte) error {
	s.mu.RLock()
	peer, ok := s.peers[header.Id]
	s.mu.RUnlock()
	if ok {
		_, err := peer.c.Send(&subpub.UniMessage{Header: header, PayloadType: payloadType, Payload: payload})
		return err
	}
	return errPeerNotExist
}

func (s *Local) Start() {
	s.s.Serve(s.l)
}

func (s *Local) Stop() {
	s.l.Close()
	s.s.GracefulStop()
}

func (s *Local) RegisterCallback(dest Identity, cb LocalCb) {
	s.mu.Lock()
	s.localCbs[dest] = cb
	s.mu.Unlock()
}

func NewLocal(lis net.Listener) *Local {
	l := &Local{
		l:        lis,
		localCbs: make(map[Identity]LocalCb),
	}
	l.s = stream.NewServer(l.recv, stream.Simple{})
	return l
}
