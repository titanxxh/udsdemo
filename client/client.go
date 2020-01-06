package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/stream"
)

func recvPack(c *stream.Conn, p stream.Packet) {
	fmt.Println("Client recv:", proto.CompactTextString(p.(*subpub.ServerMessage)))
}

func sendLoop(myc *stream.Conn, gracefulStop <-chan struct{}) {
	payload := make([]byte, 4096)
	msg := &subpub.ClientMessage{Header: &subpub.Header{Id: uint64(time.Now().UnixNano())}, Payload: payload}
	for {
		select {
		case <-gracefulStop:
			myc.Stop()
			fmt.Println("Client exit: final header", msg.Header)
			return
		default:
			msg.Header.Generation++
			_, err := myc.Send(msg)
			if err != nil {
				myc.Stop()
				log.Fatal("Write error:", err)
				return
			}
			fmt.Println("Client sent:", msg.Header)
		}
	}
}

func main() {
	c, err := net.Dial("unix", "/tmp/go.sock")
	if err != nil {
		log.Fatal("Dial error", err)
	}
	defer c.Close()

	myc := stream.NewConn(c, recvPack, stream.Protobuf{})
	go myc.RecvLoop()
	gracefulStop := make(chan struct{})
	go sendLoop(myc, gracefulStop)
	time.Sleep(time.Millisecond * 100)
	close(gracefulStop)
}
