package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/stream"
)

func ackPack(c *stream.Conn, p stream.Packet) {
	x := p.(*subpub.ClientMessage)
	y := &subpub.ServerMessage{Header: x.GetHeader()}
	_, err := c.Send(y)
	if err != nil {
		log.Printf("Server resp error: %v", err)
	}
	fmt.Println("Server send:", y)
}

func main() {
	os.RemoveAll("/tmp/go.sock")
	log.Println("Starting echo server")
	ln, err := net.Listen("unix", "/tmp/go.sock")
	if err != nil {
		log.Fatal("Listen error: ", err)
	}
	server := stream.NewServer(ackPack, stream.Protobuf{})

	done := make(chan struct{})
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func(ln net.Listener, c chan os.Signal) {
		sig := <-c
		log.Printf("Caught signal %s: shutting down.", sig)
		ln.Close()
		server.GracefulStop()
		done <- struct{}{}
	}(ln, sigc)
	go server.Serve(ln)
	<-done
}
