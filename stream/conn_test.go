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

func ackPack(c *Conn, p *subpub.UniMessage) {
	y := &subpub.UniMessage{Header: p.GetHeader()}
	_, err := c.Send(y)
	if err != nil {
		log.Printf("Server resp error: %v", err)
	}
	fmt.Println("Server send:", y)
}

func recvPack(c *Conn, p *subpub.UniMessage) {
	fmt.Println("Client recv:", proto.CompactTextString(p))
}

func TestConn_Send(t *testing.T) {
	Convey("all", t, func() {
		bufferSize := 1024
		lis := bufconn.Listen(bufferSize)
		noti := make(chan int)
		server := NewServer(func(c *Conn, p Packet) {
			ackPack(c, p.(*subpub.UniMessage))
			noti <- 1
		}, Simple{})
		go server.Serve(lis)
		conn, err := lis.Dial()
		if err != nil {
			panic(err)
		}
		cli := NewConn(conn, func(c *Conn, p Packet) {
			recvPack(c, p.(*subpub.UniMessage))
			noti <- 2
		}, Simple{})
		go cli.RecvLoop()
		payload := make([]byte, 4096)
		msg := &subpub.UniMessage{Header: &subpub.Header{Id: uint64(time.Now().UnixNano())}, Payload: payload}
		cli.Send(msg)
		So(<-noti, ShouldEqual, 1)
		So(<-noti, ShouldEqual, 2)
		cli.Stop()
		lis.Close()
		server.GracefulStop()
	})
}
