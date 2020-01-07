package stub

import (
	"net"
	"os"
	"runtime"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"xuxinhao.com/pbsocket/mlog"
)

const (
	testAgentIden = 1

	testSubReqType    = 1
	testSubRspType    = 2
	testNormalReqType = 3
)

var (
	subReq = Message{
		From:        testAgentIden,
		To:          testAgentIden,
		PayloadType: testSubReqType,
	}
	subRsp = Message{
		From:        testAgentIden,
		To:          testAgentIden,
		PayloadType: testSubRspType,
	}
)

type serverCb struct {
	rsp *Server
}

func (s *serverCb) OnPayloadRecv(remote PeerID, msg Message) {
	mlog.L.Infof("server recv from %v, %+v", remote, msg)
	err := s.rsp.Response(remote, subRsp)
	mlog.L.Info("server reply result: ", err)
}

func (s *serverCb) OnPeerReconnect(id PeerID, gene Gene) {
}

type clientCb struct {
	req *Client
}

func (c *clientCb) OnPayloadRecv(remote PeerID, msg Message) {
	mlog.L.Infof("client recv %+v", msg)
}

func (c *clientCb) OnPeerReconnect(id PeerID, gene Gene) {
	err := c.req.Request(subReq)
	mlog.L.Info("client resub result: ", err)
}

func TestStub(t *testing.T) {
	Convey("all", t, func() {
		startNum := runtime.NumGoroutine()
		sockFile := "/tmp/go.sock"
		os.RemoveAll(sockFile)
		ln, err := net.Listen("unix", sockFile)
		if err != nil {
			mlog.L.Fatal("Listen error: ", err)
		}
		server := NewServer(ln, 1, 1)
		go server.Start()
		defer func() {
			server.Stop()
			So(runtime.NumGoroutine(), ShouldEqual, startNum)
		}()
		mlog.L.Info("server started")
		time.Sleep(time.Second)
		client := NewClient(func() (conn net.Conn, err error) {
			return net.Dial("unix", sockFile)
		}, 1, 1)
		scb, ccb := &serverCb{rsp: server}, &clientCb{req: client}
		server.RegisterCallback(1, scb)
		client.RegisterCallback(1, ccb)
		client.Request(subReq)
		mlog.L.Info("client sub result: ", err)
		time.Sleep(time.Second)

		// for connection lost then send request cause re-dial again
		client.conn.Stop()
		So(client.Request(subReq), ShouldBeError)
		time.Sleep(time.Second)
	})
}
