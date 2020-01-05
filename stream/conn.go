package stream

import (
	"bytes"
	"io"
	"log"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
)

const RecvBufSize = 1024

type OnRecvPack func(c net.Conn, p proto.Message) 

// Conn ...
type Conn struct {
	c net.Conn
	handler OnRecvPack
}

func (c Conn) RecvLoop() {
	var tempDelay time.Duration
	maxDelay := 1 * time.Second
	tempBuf := make([]byte, RecvBufSize)
	recvBuf := bytes.NewBuffer(make([]byte, 0, RecvBufSize))
	pr := Protocol{}
	for {
		n, err := c.c.Read(tempBuf)
		if err != nil {
			// unix domain will produce a net error?
			if nerr, ok := err.(net.Error); ok {
				if nerr.Timeout() {
					// timeout
				} else if nerr.Temporary() {
					if tempDelay == 0 {
						tempDelay = 5 * time.Millisecond
					} else {
						tempDelay *= 2
					}
					if tempDelay > maxDelay {
						tempDelay = maxDelay
					}
					log.Printf("Conn[%v] recv error: %v; retrying in %v", c.c.RemoteAddr(), err, tempDelay)
					time.Sleep(tempDelay)
					continue
				}
			}
			if err != io.EOF {
				log.Printf("Conn[%v] recv error: %v", c.c.RemoteAddr(), err)
			} else {
				c.c.Close()
			}
			return
		}

		recvBuf.Write(tempBuf[:n])
		tempDelay = 0

		for recvBuf.Len() > 0 {
			p, pl, err := pr.Unpack(recvBuf.Bytes())
			if err != nil {
				log.Printf("Conn[%v] OnUnpackErr: %v, bytes: %v", c.c.RemoteAddr(), err, recvBuf.Bytes())
			}
			if pl > 0 {
				_ = recvBuf.Next(pl)
			}
			if p != nil {
				c.handler(c.c, p)
			} else {
				break
			}
		}
	}
}

func NewConn(c net.Conn, cb OnRecvPack) Conn {
	return Conn{c: c, handler: cb}
}
