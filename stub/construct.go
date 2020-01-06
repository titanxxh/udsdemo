package stub

import "xuxinhao.com/pbsocket/api/subpub"

func ConstructHelloAck(selfId uint64, gene int32) *subpub.UniMessage {
	return &subpub.UniMessage{
		Header:      &subpub.Header{Id: selfId, Generation: gene, From: 0, To: 0},
		PayloadType: HelloAck,
	}
}

func ConstructGoodbyeAck(selfId uint64, gene int32) *subpub.UniMessage {
	return &subpub.UniMessage{
		Header:      &subpub.Header{Id: selfId, Generation: gene, From: 0, To: 0},
		PayloadType: GoodbyeAck,
	}
}
