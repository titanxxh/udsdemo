package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/protobuf/proto"
	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/stream"
)

var pr = stream.Protocol{}

func ackPack(c net.Conn, p proto.Message) {
	fmt.Println(proto.CompactTextString(p))
	x := p.(*subpub.ClientMessage)
	y := &subpub.ServerMessage{Header: x.GetHeader()}
	_, err := pr.PackTo(y, c)
	if err != nil {
		log.Printf("Server resp error: %v", err)
	}
}

func echoServer(c net.Conn) {
	log.Printf("Accept conn %s: %p", c.RemoteAddr().String(), c)
	defer func() {
		log.Printf("Conn[%v] exit", c.RemoteAddr())
	}()
	myc := stream.NewConn(c, ackPack)
	myc.RecvLoop()
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

		go echoServer(fd)
	}
}
