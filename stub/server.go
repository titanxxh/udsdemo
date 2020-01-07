package stub

import (
	"fmt"
	"net"
	"sync"

	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/mlog"
	"xuxinhao.com/pbsocket/stream"
)

var (
	errPeerNotExist = fmt.Errorf("peer not exist")
)

type (
	// todo maybe we must use another unique process id rather than a uint64.
	PeerID = uint64
	// a random number when process is started.
	Gene = int32
	// an identity for a component
	Identity = int32
	Message  struct {
		// identity for a component
		From, To    int32
		PayloadType int32
		Payload     []byte
	}
	Handler interface {
		// a message from remote process, will redirect to the 'To' identity in header.
		// to server side:
		// remote is PeerID of the client.
		// to client side:
		// remote is PeerID of the server, should never change.
		OnPayloadRecv(remote PeerID, msg Message)
		// to server side, this handler is called when:
		// the client process is reconnected (id not changed but generation changed), usually do nothing.
		// to client side, this handler is called when:
		// the connection just setup or re-setup, usually re-subscribe all the subscriptions of server.
		OnPeerReconnect(id PeerID)
	}
	peerInfo struct {
		id   PeerID
		gene Gene
		c    *stream.Conn
	}
)

const (
	Hello = iota
	HelloAck
	Goodbye
	GoodbyeAck
)

type (
	Server struct {
		s        *stream.Server
		l        net.Listener
		mu       sync.RWMutex
		cbs      map[Identity]Handler
		peers    map[PeerID]peerInfo
		self     PeerID
		selfGene Gene
	}
)

func (s *Server) hello(peer peerInfo) {
	s.mu.Lock()
	old, ok := s.peers[peer.id]
	if ok {
		if old.gene == peer.gene {
			if old.c != peer.c {
				// if true, raw connection changed means network error?
				old.c.Stop()
				s.peers[peer.id] = peer
			}
			// if not, a duplicated hello usually means the hello ack failed
			s.mu.Unlock()
			peer.c.Send(ConstructHelloAck(s.self, s.selfGene))
			return
		}
		old.c.Stop()
		mlog.L.Debugf("Server: old client %v-%v stop", old.id, old.gene)
		for _, cb := range s.cbs {
			cb.OnPeerReconnect(peer.id)
		}
	}
	s.peers[peer.id] = peer
	mlog.L.Debugf("Server: new client %v-%v added", peer.id, peer.gene)
	s.mu.Unlock()
	peer.c.Send(ConstructHelloAck(s.self, s.selfGene))
}

func (s *Server) goodbye(peer peerInfo) {
	s.mu.Lock()
	p, ok := s.peers[peer.id]
	if ok {
		delete(s.peers, peer.id)
		mlog.L.Debugf("Server: client %v-%v delete", p.id, p.gene)
		peer.c.Send(ConstructGoodbyeAck(s.self, s.selfGene))
	}
	s.mu.Unlock()
}

func (s *Server) internal(peer peerInfo, t int32, buf []byte) {
	switch t {
	case Hello:
		s.hello(peer)
	case Goodbye:
		s.goodbye(peer)
	}
}

func (s *Server) recv(c *stream.Conn, pack stream.Packet) {
	m := pack.(*subpub.UniMessage)
	if m.Header == nil {
		return
	}
	to := m.Header.To
	if to == 0 {
		s.internal(peerInfo{id: m.Header.Id, gene: m.Header.Generation, c: c}, m.PayloadType, m.Payload)
		return
	}
	s.mu.RLock()
	cb, ok := s.cbs[to]
	s.mu.RUnlock()
	if !ok {
		return
	}
	cb.OnPayloadRecv(m.Header.Id, Message{From: m.Header.From, To: m.Header.To, Payload: m.Payload, PayloadType: m.PayloadType})
}

func (s *Server) Response(peerId PeerID, m Message) error {
	s.mu.RLock()
	peer, ok := s.peers[peerId]
	s.mu.RUnlock()
	if ok {
		h := ConstructHeader(s.self, s.selfGene, m.From, m.To)
		_, err := peer.c.Send(&subpub.UniMessage{Header: h, PayloadType: m.PayloadType, Payload: m.Payload})
		return err
	}
	return errPeerNotExist
}

func (s *Server) Start() {
	s.s.Serve(s.l)
}

func (s *Server) Stop() {
	s.l.Close()
	s.s.GracefulStop()
}

// dest is the identity for the destination component
func (s *Server) RegisterCallback(dest Identity, cb Handler) {
	s.mu.Lock()
	s.cbs[dest] = cb
	s.mu.Unlock()
}

// NewServer...
func NewServer(lis net.Listener, self PeerID, selfGene Gene) *Server {
	l := &Server{
		l:        lis,
		cbs:      make(map[Identity]Handler),
		peers:    make(map[PeerID]peerInfo),
		self:     self,
		selfGene: selfGene,
	}
	l.s = stream.NewServer(l.recv, stream.Simple{})
	return l
}
