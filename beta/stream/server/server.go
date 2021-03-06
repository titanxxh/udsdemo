package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/mlog"
	"xuxinhao.com/pbsocket/stream"
)

func ackPack(c *stream.Conn, p stream.Packet) {
	x := p.(*subpub.ClientMessage)
	y := &subpub.ServerMessage{Header: x.GetHeader()}
	_, err := c.Send(y)
	if err != nil {
		mlog.L.Info("Server resp error: %v", err)
	}
	mlog.L.Info("Server send:", y)
}

func main() {
	os.RemoveAll("/tmp/go.sock")
	mlog.L.Info("Starting echo server")
	ln, err := net.Listen("unix", "/tmp/go.sock")
	if err != nil {
		mlog.L.Fatal("Listen error: ", err)
	}
	server := stream.NewServer(ackPack, stream.Protobuf{})

	done := make(chan struct{})
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func(ln net.Listener, c chan os.Signal) {
		sig := <-c
		mlog.L.Infof("Caught signal %s: shutting down.", sig)
		ln.Close()
		server.GracefulStop()
		done <- struct{}{}
	}(ln, sigc)
	go server.Serve(ln)
	<-done
	mlog.L.Infof("stat: %+v", server.GetCurrentStat())
}
