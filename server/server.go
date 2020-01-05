package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/stream"
)

const RecvBufSize = 1024

func recvPack(c net.Conn, p stream.Packet) {
	fmt.Println(p.String())
}

func echoServer(c net.Conn) {
	defer func() {
		log.Printf("Conn[%v] exit", c.RemoteAddr())
	}()
	var tempDelay time.Duration
	maxDelay := 1 * time.Second
	tempBuf := make([]byte, RecvBufSize)
	recvBuf := bytes.NewBuffer(make([]byte, 0, RecvBufSize))
	pr := stream.ProtobufProtocol{}
	for {
		n, err := c.Read(tempBuf)
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
					log.Printf("Conn[%v] recv error: %v; retrying in %v", c.RemoteAddr(), err, tempDelay)
					time.Sleep(tempDelay)
					continue
				}
			}
			if err != io.EOF {
				log.Printf("Conn[%v] recv error: %v", c.RemoteAddr(), err)
			} else {
				c.Close()
			}
			return
		}

		recvBuf.Write(tempBuf[:n])
		tempDelay = 0

		for recvBuf.Len() > 0 {
			p, pl, err := pr.Unpack(recvBuf.Bytes())
			if err != nil {
				log.Printf("Conn[%v] OnUnpackErr: %v, bytes: %v", c.RemoteAddr(), err, recvBuf.Bytes())
			}
			if pl > 0 {
				_ = recvBuf.Next(pl)
			}
			if p != nil {
				recvPack(c, p)
			} else {
				break
			}
		}
	}
}

func main() {
	log.Println("Starting echo server")
	ln, err := net.Listen("unix", "/tmp/go.sock")
	if err != nil {
		log.Fatal("Listen error: ", err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func(ln net.Listener, c chan os.Signal) {
		sig := <-c
		log.Printf("Caught signal %s: shutting down.", sig)
		ln.Close()
		os.Exit(0)
	}(ln, sigc)

	for {
		fd, err := ln.Accept()
		if err != nil {
			log.Fatal("Accept error: ", err)
		}
		log.Printf("Accept conn %s: %p", fd.RemoteAddr().String(), fd)

		go echoServer(fd)
	}
}
