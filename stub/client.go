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
	errConn = fmt.Errorf("connection lost, please retry later")

	retryInterval  = time.Second
	waitAckMaxTime = time.Millisecond * 500
)

type (
	DialFunc  func() (net.Conn, error)
	connState = int32
)

const (
	notConnected connState = iota
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
	ctrlChan chan connState
}

func (r *Client) GetCurrentStat() stream.Statistics {
	r.mu.RLock()
	t := r.stat
	if r.conn != nil {
		t = stream.AddStat(t, r.conn.GetCurrentStat())
	}
	r.mu.RUnlock()
	return t
}

func (r *Client) retryDial(wait time.Duration) {
	atomic.StoreInt32(&r.state, notConnected)
	if wait > 0 {
		mlog.L.Infof("Client %v-%v retry dial later", r.self, r.selfGene)
		time.AfterFunc(wait, r.tryDial)
	} else {
		mlog.L.Infof("Client %v-%v immediately retry dial", r.self, r.selfGene)
		go r.tryDial()
	}
}

func (r *Client) tryDial() {
	d, ok := r.dial()
	if !ok {
		r.retryDial(d)
	}
}

func (r *Client) dial() (time.Duration, bool) {
	mlog.L.Debugf("Client %v-%v dial", r.self, r.selfGene)
	if atomic.LoadInt32(&r.state) != notConnected {
		return 0, true
	}
	r.mu.Lock()
	r.conn = nil
	conn, err := r.dialFunc()
	if err != nil {
		r.mu.Unlock()
		mlog.L.Debugf("Client %v-%v dial failed %v", r.self, r.selfGene, err)
		return retryInterval, false
	}
	sConn := stream.NewConn(conn, r.recv, stream.Simple{})
	r.conn = sConn
	r.mu.Unlock()

	r.mu.RLock()
	defer r.mu.RUnlock()
	_, err = r.conn.Send(ConstructHello(r.self, r.selfGene))
	if err != nil {
		r.conn.Stop()
		mlog.L.Debugf("Client %v-%v dial hello failed %v. Stop connection", r.self, r.selfGene, err)
		return retryInterval, false
	}
	go func() {
		mlog.L.Infof("Client %v-%v recv start", r.self, r.selfGene)
		sConn.RecvLoop()
		mlog.L.Infof("Client %v-%v recv exit, %+v", r.self, r.selfGene, sConn.GetCurrentStat())
		r.mu.Lock()
		r.stat = stream.AddStat(r.stat, sConn.GetCurrentStat())
		// if exit loop from connected, means hello ack is received, retry immediately
		// otherwise, retry later
		if atomic.LoadInt32(&r.state) == connected {
			r.retryDial(0)
		} else {
			r.retryDial(retryInterval)
		}
		r.mu.Unlock()
	}()
	wa := time.NewTimer(waitAckMaxTime)
	select {
	case <-wa.C:
		r.conn.Stop()
		mlog.L.Infof("Client %v-%v dial wait hello ack timeout. Stop connection", r.self, r.selfGene)
	case <-r.ctrlChan:
		wa.Stop()
		return 0, true
	}
	return retryInterval, false
}

func (r *Client) internal(peer peerInfo, t int32, buf []byte) {
	switch t {
	case HelloAck:
		if atomic.CompareAndSwapInt32(&r.state, notConnected, connected) {
			mlog.L.Infof("Client %v-%v change to connected", r.self, r.selfGene)
			r.mu.RLock()
			for _, cb := range r.cbs {
				cb.OnPeerReconnect(peer.id)
			}
			r.mu.RUnlock()
			select {
			case r.ctrlChan <- connected:
				mlog.L.Infof("Client %v-%v notify hello ack.", r.self, r.selfGene)
			default:
			}
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
	if err != nil {
		r.conn.Stop()
		mlog.L.Debugf("Client %v-%v request ptype %v head %v failed, %v. Stop connection", r.self, r.selfGene, m.PayloadType, h, err)
	}
	r.mu.RUnlock()
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
		ctrlChan: make(chan connState, 1),
	}
	l.tryDial()
	return l
}
