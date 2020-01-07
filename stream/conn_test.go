package stream

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc/test/bufconn"
	"xuxinhao.com/pbsocket/api/subpub"
	"xuxinhao.com/pbsocket/mlog"
)

func ackPack(c *Conn, p *subpub.UniMessage) {
	y := &subpub.UniMessage{Header: p.GetHeader()}
	_, err := c.Send(y)
	if err != nil {
		mlog.L.Infof("Server resp error: %v", err)
	}
	mlog.L.Info("Server send:", y)
}

func recvPack(c *Conn, p *subpub.UniMessage) {
	mlog.L.Info("Client recv:", proto.CompactTextString(p))
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
		x := 0
		x |= 1 << (<-noti - 1)
		x |= 1 << (<-noti - 1)
		// client and server goroutines are parallel
		So(x, ShouldEqual, 3)
		cli.Stop()
		lis.Close()
		server.GracefulStop()
	})
}

func TestXX_Manytimes(t *testing.T) {
	for i := 0; i < 0; i++ {
		mlog.L = mlog.L.WithField("times", i)
		TestConn_Send(t)
	}
}
