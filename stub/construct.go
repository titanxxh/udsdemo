package stub

import "xuxinhao.com/pbsocket/api/subpub"

func ConstructHeader(selfId PeerID, gene Gene, from, to Identity) *subpub.Header {
	return &subpub.Header{Id: selfId, Generation: gene, From: from, To: to}
}

func ConstructHello(selfId PeerID, gene Gene) *subpub.UniMessage {
	return &subpub.UniMessage{
		Header:      ConstructHeader(selfId, gene, 0, 0),
		PayloadType: Hello,
	}
}

func ConstructHelloAck(selfId PeerID, gene Gene) *subpub.UniMessage {
	return &subpub.UniMessage{
		Header:      ConstructHeader(selfId, gene, 0, 0),
		PayloadType: HelloAck,
	}
}

func ConstructGoodbye(selfId PeerID, gene Gene) *subpub.UniMessage {
	return &subpub.UniMessage{
		Header:      ConstructHeader(selfId, gene, 0, 0),
		PayloadType: Goodbye,
	}
}

func ConstructGoodbyeAck(selfId PeerID, gene Gene) *subpub.UniMessage {
	return &subpub.UniMessage{
		Header:      ConstructHeader(selfId, gene, 0, 0),
		PayloadType: GoodbyeAck,
	}
}
