package itf

import "xuxinhao.com/pbsocket/stub"

func constructBuf(n int) []byte {
	x := make([]byte, n)
	for i := 0; i < n; i++ {
		x[i] = 1
	}
	return x
}

var (
	Msg256 = stub.Message{From: TestAgentIden, To: TestAgentIden, PayloadType: TestNormalReqType, Payload: constructBuf(256)}
	Msg1K  = stub.Message{From: TestAgentIden, To: TestAgentIden, PayloadType: TestNormalReqType, Payload: constructBuf(1024)}
	Msg4K  = stub.Message{From: TestAgentIden, To: TestAgentIden, PayloadType: TestNormalReqType, Payload: constructBuf(4 * 1024)}
	Msg1M  = stub.Message{From: TestAgentIden, To: TestAgentIden, PayloadType: TestNormalReqType, Payload: constructBuf(1024 * 1024)}
)

const (
	TestAgentIden = 1

	TestSubReqType    = 1
	TestSubRspType    = 2
	TestNormalReqType = 3
	TestNormalRspType = 4
)

var (
	SubReq = stub.Message{
		From:        TestAgentIden,
		To:          TestAgentIden,
		PayloadType: TestSubReqType,
	}
	SubRsp = stub.Message{
		From:        TestAgentIden,
		To:          TestAgentIden,
		PayloadType: TestSubRspType,
	}
)
