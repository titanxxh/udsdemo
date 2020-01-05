package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/stream"
)

func reader(r io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf[:])
		if err != nil {
			return
		}
		println("Client got:", string(buf[0:n]))
	}
}

func main() {
	c, err := net.Dial("unix", "/tmp/go.sock")
	if err != nil {
		log.Fatal("Dial error", err)
	}
	defer c.Close()

	go reader(c)
	msg := &subpub.ClientMessage{Header: &subpub.Header{Id: uint64(time.Now().UnixNano())}}
	pr := stream.ProtobufProtocol{}
	for {
		msg.Header.Generation++
		sm := stream.NewProtobufPacket(msg)
		_, err := pr.PackTo(sm, c)
		if err != nil {
			log.Fatal("Write error:", err)
			break
		}
		fmt.Println("Client sent:", sm.String())
		if msg.Header.Generation % 10 == 0 {
			time.Sleep(time.Second)
		}
	}
}
