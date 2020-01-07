package stream

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"xuxinhao.com/pbsocket/mlog"
)

var (
	errSendToClosedConn = fmt.Errorf("send to closed conn")
)

const (
	normal = iota
	stopped
)

// RecvBufSize ...
const RecvBufSize = 1024

// Conn wraps raw net.Conn
// We inject a OnPackReceived func to handle callback.
type Conn struct {
	c       net.Conn
	handler OnPackReceived
	state   int32
	stop    sync.Once
	l       sync.Mutex
	pr      Protocol
	stat    Statistics
}

func (c *Conn) GetCurrentStat() Statistics {
	c.l.Lock()
	t := c.stat
	c.l.Unlock()
	return t
}

func (c *Conn) isStopped() bool {
	return atomic.LoadInt32(&c.state) == stopped
}

func (c *Conn) isRunning() bool {
	return atomic.LoadInt32(&c.state) == normal
}

// Stop stops the conn.
func (c *Conn) Stop() {
	c.stop.Do(func() {
		atomic.StoreInt32(&c.state, stopped)
		c.c.Close()
	})
}

// Send send a pack.
func (c *Conn) Send(msg Packet) (int, error) {
	if !c.isRunning() {
		return 0, errSendToClosedConn
	}
	c.l.Lock()
	n, err := c.pr.PackInto(msg, c.c)
	c.stat.ByteSend += uint64(n)
	if err != nil {
		c.stat.PacketSendErr++
	} else {
		c.stat.PacketSend++
	}
	c.l.Unlock()
	return n, err
}

// RecvLoop will run until the connection is stopped or read an error.
func (c *Conn) RecvLoop() {
	defer func() {
		mlog.L.Debugf("Conn[%v] exit recv", c.c.RemoteAddr())
	}()
	tempBuf := make([]byte, RecvBufSize)
	recvBuf := bytes.NewBuffer(make([]byte, 0, RecvBufSize))
	for {
		n, err := c.c.Read(tempBuf)
		if err != nil {
			if c.isRunning() {
				if err != io.EOF {
					mlog.L.Debugf("Conn[%v] recv error cause stop: %v", c.c.RemoteAddr(), err)
				} else {
					mlog.L.Debugf("Conn[%v] recv io eof", c.c.RemoteAddr())
				}
				c.Stop()
			}
			return
		}

		recvBuf.Write(tempBuf[:n])
		for recvBuf.Len() > 0 {
			msg, upl, err := c.pr.Unpack(recvBuf.Bytes())
			if err != nil {
				mlog.L.Errorf("Conn[%v] unpack error: %v, bytes: %v", c.c.RemoteAddr(), err, recvBuf.Bytes())
				c.stat.PacketRecvErr++
				// todo if err count too much, abort this conn?
			}
			if upl > 0 {
				_ = recvBuf.Next(upl)
			}
			c.stat.ByteRecv += uint64(upl)
			if msg != nil {
				c.handler(c, msg)
				c.stat.PacketRecv++
			} else {
				break
			}
		}
	}
}

// NewConn ...
func NewConn(c net.Conn, cb OnPackReceived, pr Protocol) *Conn {
	return &Conn{c: c, handler: cb, pr: pr}
}
