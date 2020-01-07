package stub

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/mlog"
	"xuxinhao.com/pbsocket/stream"
)

var (
	errConn = fmt.Errorf("connection error, please retry later")

	retryInterval = time.Second
)

type (
	DialFunc  func() (net.Conn, error)
	connState = int32
)

const (
	notDailed connState = iota
	dailFailed
	dailed
	waitHelloAck
	connected
)

type Client struct {
	mu       sync.RWMutex
	self     PeerID
	selfGene Gene
	dialFunc DialFunc
	conn     *stream.Conn
	cbs      map[Identity]Handler
	state    connState
	stat     stream.Statistics
}

func (r *Client) GetCurrentStat() stream.Statistics {
	r.mu.RLock()
	t := r.stat
	r.mu.RUnlock()
	t = stream.AddStat(t, r.conn.GetCurrentStat())
	return t
}

func (r *Client) retryDail(old connState, wait time.Duration) {
	sw := atomic.CompareAndSwapInt32(&r.state, old, dailFailed)
	if sw {
		if wait > 0 {
			time.AfterFunc(wait, func() {
				atomic.StoreInt32(&r.state, notDailed)
				r.tryDial()
			})
		} else {
			go func() {
				atomic.StoreInt32(&r.state, notDailed)
				r.tryDial()
			}()
		}
	}
}

func (r *Client) tryDial() {
	mlog.L.Debugf("Client %v-%v try dial", r.self, r.selfGene)
	s := atomic.LoadInt32(&r.state)
	if s != notDailed {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conn = nil
	conn, err := r.dialFunc()
	if err != nil {
		atomic.StoreInt32(&r.state, dailFailed)
		mlog.L.Debugf("Client %v-%v try dial failed %v", r.self, r.selfGene, err)
		r.retryDail(s, retryInterval)
		return
	}
	atomic.StoreInt32(&r.state, dailed)
	sConn := stream.NewConn(conn, r.recv, stream.Simple{})
	go func() {
		mlog.L.Infof("Client %v-%v recv start", r.self, r.selfGene)
		sConn.RecvLoop()
		mlog.L.Infof("Client %v-%v recv exit, %+v", r.self, r.selfGene, sConn.GetCurrentStat())
		r.mu.Lock()
		r.stat = stream.AddStat(r.stat, sConn.GetCurrentStat())
		r.mu.Unlock()
	}()
	r.conn = sConn
	_, err = r.conn.Send(ConstructHello(r.self, r.selfGene))
	if err != nil {
		r.conn.Stop()
		mlog.L.Debugf("Client %v-%v try dial hello failed %v", r.self, r.selfGene, err)
		r.retryDail(dailed, retryInterval)
		return
	}
	atomic.StoreInt32(&r.state, waitHelloAck)
}

func (r *Client) internal(peer peerInfo, t int32, buf []byte) {
	switch t {
	case HelloAck:
		sw := atomic.CompareAndSwapInt32(&r.state, waitHelloAck, connected)
		if sw {
			mlog.L.Debugf("Client %v-%v change to connected", r.self, r.selfGene)
			r.mu.RLock()
			for _, cb := range r.cbs {
				cb.OnPeerReconnect(peer.id)
			}
			r.mu.RUnlock()
		}
	case GoodbyeAck:
	}
}

func (r *Client) recv(c *stream.Conn, pack stream.Packet) {
	m := pack.(*subpub.UniMessage)
	if m.Header == nil {
		return
	}
	to := m.Header.To
	if to == 0 {
		r.internal(peerInfo{id: m.Header.Id, gene: m.Header.Generation, c: c}, m.PayloadType, m.Payload)
		return
	}
	if atomic.LoadInt32(&r.state) != connected {
		return
	}
	r.mu.RLock()
	cb, ok := r.cbs[to]
	r.mu.RUnlock()
	if !ok {
		return
	}
	cb.OnPayloadRecv(m.Header.Id, Message{From: m.Header.From, To: m.Header.To, Payload: m.Payload, PayloadType: m.PayloadType})
}

// dest is the identity for the destination component
func (r *Client) Request(m Message) error {
	if atomic.LoadInt32(&r.state) != connected {
		return errConn
	}
	r.mu.RLock()
	if r.conn == nil {
		return errConn
	}
	h := ConstructHeader(r.self, r.selfGene, m.From, m.To)
	_, err := r.conn.Send(&subpub.UniMessage{
		Header:      h,
		PayloadType: m.PayloadType, Payload: m.Payload,
	})
	r.mu.RUnlock()
	if err != nil {
		r.conn.Stop()
		mlog.L.Debugf("Client %v-%v request ptype %v head %v failed, %v", r.self, r.selfGene, m.PayloadType, h, err)
		r.retryDail(connected, 0)
	}
	return err
}

func (r *Client) RegisterCallback(dest Identity, cb Handler) {
	r.mu.Lock()
	r.cbs[dest] = cb
	r.mu.Unlock()
}

// NewClient...
func NewClient(dialFunc DialFunc, self PeerID, selfGene Gene) *Client {
	l := &Client{
		dialFunc: dialFunc,
		cbs:      make(map[Identity]Handler),
		self:     self,
		selfGene: selfGene,
	}
	l.tryDial()
	return l
}
