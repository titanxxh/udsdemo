package stream

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc/test/bufconn"
	"xuxinhao.com/pbsocket/api/subpub"
)

func ackPack(c *Conn, p proto.Message) {
	x := p.(*subpub.ClientMessage)
	y := &subpub.ServerMessage{Header: x.GetHeader()}
	_, err := c.Send(y)
	if err != nil {
		log.Printf("Server resp error: %v", err)
	}
	fmt.Println("Server send:", y)
}

func recvPack(c *Conn, p proto.Message) {
	fmt.Println("Client recv:", proto.CompactTextString(p))
}

func TestConn_Send(t *testing.T) {
	Convey("all", t, func() {
		bufferSize := 1024
		lis := bufconn.Listen(bufferSize)
		noti := make(chan int)
		server := NewServer(func(c *Conn, p proto.Message) {
			ackPack(c, p)
			noti <- 1
		})
		go server.Serve(lis)
		conn, err := lis.Dial()
		if err != nil {
			panic(err)
		}
		cli := NewConn(conn, func(c *Conn, p proto.Message) {
			recvPack(c, p)
			noti <- 2
		})
		go cli.RecvLoop()
		payload := make([]byte, 4096)
		msg := &subpub.ClientMessage{Header: &subpub.Header{Id: uint64(time.Now().UnixNano())}, Payload: payload}
		cli.Send(msg)
		So(<-noti, ShouldEqual, 1)
		So(<-noti, ShouldEqual, 2)
		cli.Stop()
		lis.Close()
		server.GracefulStop()
	})
}
