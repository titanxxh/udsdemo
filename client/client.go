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

func recvPack(c net.Conn, p proto.Message) {
	fmt.Println(proto.CompactTextString(p))
}

func reader(c net.Conn) {
	log.Printf("Accept conn %s: %p", c.RemoteAddr().String(), c)
	defer func() {
		log.Printf("Conn[%v] exit", c.RemoteAddr())
	}()
	myc := stream.NewConn(c, recvPack)
	myc.RecvLoop()
}

func main() {
	c, err := net.Dial("unix", "/tmp/go.sock")
	if err != nil {
		log.Fatal("Dial error", err)
	}
	defer c.Close()

	go reader(c)
	msg := &subpub.ClientMessage{Header: &subpub.Header{Id: uint64(time.Now().UnixNano())}}
	pr := stream.Protocol{}
	for {
		msg.Header.Generation++
		_, err := pr.PackTo(msg, c)
		if err != nil {
			log.Fatal("Write error:", err)
			break
		}
		fmt.Println("Client sent:", msg)
		if msg.Header.Generation % 10 == 0 {
			time.Sleep(time.Second)
		}
	}
}
