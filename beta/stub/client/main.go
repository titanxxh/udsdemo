package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"xuxinhao.com/pbsocket/beta/stub/itf"
	"xuxinhao.com/pbsocket/mlog"
	"xuxinhao.com/pbsocket/stream"
	"xuxinhao.com/pbsocket/stub"
)

type haf struct {
	c *stub.Client
}

func (h *haf) OnPayloadRecv(remote stub.PeerID, msg stub.Message) {
}

func (h *haf) OnPeerReconnect(id stub.PeerID) {
}

func (h *haf) sub(key int) {
	h.c.Request(itf.SubReq)
}

func main() {
	rs := rand.NewSource(time.Now().UnixNano())
	gene := stub.Gene(rs.Int63())
	fmt.Println(os.Args)
	s, _ := strconv.ParseInt(os.Args[1], 10, 32)
	self := stub.PeerID(s)
	client := stub.NewClient(func() (conn net.Conn, err error) {
		return net.Dial("unix", "/tmp/go.sock")
	}, self, gene)
	h := &haf{c: client}
	client.RegisterCallback(itf.TestAgentIden, h)
	perSec, _ := strconv.ParseInt(os.Args[2], 10, 32)
	mlog.L.Infof("ready")
	go func() {
		prev, pStat := time.Now(), client.GetCurrentStat()
		for {
			for i := int64(0); i < perSec; i++ {
				client.Request(itf.Msg256)
			}
			time.Sleep(time.Second)
			curr, cStat := time.Now(), client.GetCurrentStat()
			sec := curr.Sub(prev).Seconds()
			delta := stream.SubStat(cStat, pStat)
			if sec > 0 {
				mlog.L.Infof("client stat: %+v", stream.StatPerSec(delta, sec))
			}
			prev, pStat = curr, cStat
		}
	}()

	done := make(chan struct{})
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		mlog.L.Info("Caught signal %s: shutting down.", sig)
		mlog.L.Infof("client stat: %+v", client.GetCurrentStat())
		done <- struct{}{}
	}(sigc)
	<-done
}
