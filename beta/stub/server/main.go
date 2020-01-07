package main

import (
	"encoding/binary"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xuxinhao.com/pbsocket/beta/stub/itf"
	"xuxinhao.com/pbsocket/mlog"
	"xuxinhao.com/pbsocket/stream"
	"xuxinhao.com/pbsocket/stub"
)

type haf struct {
	c *stub.Server
	// haf field
	clientSeq [100]uint64
}

func (h *haf) OnPayloadRecv(remote stub.PeerID, msg stub.Message) {
	seq := binary.LittleEndian.Uint64(msg.Payload[:8])
	if seq != h.clientSeq[remote]+1 {
		mlog.L.Errorf("server seq check failed, client %v before %d now %d", remote, h.clientSeq[remote], seq)
	}
	h.clientSeq[remote] = seq
	h.c.Response(remote, msg)
}

func (h *haf) OnPeerReconnect(id stub.PeerID) {
	mlog.L.Errorf("server checked a client reset, client %v", id)
}

func main() {
	rs := rand.NewSource(time.Now().UnixNano())
	gene := stub.Gene(rs.Int63())
	self := stub.PeerID(0)
	sockFile := "/tmp/go.sock"
	os.RemoveAll(sockFile)
	ln, err := net.Listen("unix", sockFile)
	if err != nil {
		mlog.L.Fatal("Listen error: ", err)
	}
	server := stub.NewServer(ln, self, gene)
	go server.Start()
	h := &haf{c: server}
	server.RegisterCallback(itf.TestAgentIden, h)
	mlog.L.Infof("ready")

	done := make(chan struct{})
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func(ln net.Listener, c chan os.Signal) {
		sig := <-c
		mlog.L.Infof("Caught signal %s: shutting down.", sig)
		ln.Close()
		server.Stop()
		done <- struct{}{}
	}(ln, sigc)
	go func() {
		ticker := time.NewTicker(time.Second)
		prev, pStat := time.Now(), server.GetCurrentStat()
		for range ticker.C {
			curr, cStat := time.Now(), server.GetCurrentStat()
			sec := curr.Sub(prev).Seconds()
			delta := stream.SubStat(cStat, pStat)
			if sec > 0 {
				mlog.L.Infof("server stat: %+v", stream.StatPerSec(delta, sec))
			}
			prev, pStat = curr, cStat
		}
	}()
	<-done
}
