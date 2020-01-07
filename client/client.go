package main

import (
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/mlog"
	"xuxinhao.com/pbsocket/stream"
)

func recvPack(c *stream.Conn, p stream.Packet) {
	mlog.L.Info("Client recv:", proto.CompactTextString(p.(*subpub.ServerMessage)))
}

func sendLoop(myc *stream.Conn, gracefulStop <-chan struct{}) {
	payload := make([]byte, 4096)
	msg := &subpub.ClientMessage{Header: &subpub.Header{Id: uint64(time.Now().UnixNano())}, Payload: payload}
	for {
		select {
		case <-gracefulStop:
			myc.Stop()
			mlog.L.Info("Client exit: final header", msg.Header)
			return
		default:
			msg.Header.Generation++
			_, err := myc.Send(msg)
			if err != nil {
				myc.Stop()
				mlog.L.Fatal("Write error:", err)
				return
			}
			mlog.L.Info("Client sent:", msg.Header)
		}
	}
}

func main() {
	c, err := net.Dial("unix", "/tmp/go.sock")
	if err != nil {
		mlog.L.Fatal("Dial error", err)
	}
	defer c.Close()

	myc := stream.NewConn(c, recvPack, stream.Protobuf{})
	go myc.RecvLoop()
	gracefulStop := make(chan struct{})
	go sendLoop(myc, gracefulStop)
	time.Sleep(time.Millisecond * 100)
	close(gracefulStop)
}
