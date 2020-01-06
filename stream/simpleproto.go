package stream

import (
	"encoding/binary"
	"io"

	"github.com/golang/protobuf/proto"
	"xuxinhao.com/pbsocket/api/subpub"
)

type Simple struct{}

func (s Simple) PackSize(pack Packet) int {
	pb, ok := pack.(*subpub.UniMessage)
	if !ok {
		return 0
	}
	return 4 + proto.Size(pb)
}

func (s Simple) PackInto(pack Packet, w io.Writer) (int, error) {
	pb, ok := pack.(*subpub.UniMessage)
	if !ok {
		return 0, errUnknownMsgType
	}
	msgLen := 4 + proto.Size(pb)

	data, err := proto.Marshal(pb)
	if err != nil {
		return 0, err
	}

	wl := 0
	err = binary.Write(w, binary.BigEndian, uint32(msgLen))
	if err != nil {
		return wl, err
	}
	wl += 4
	n, err := w.Write(data)
	wl += n
	if err != nil {
		return wl, err
	}
	return wl, nil
}

func (s Simple) Unpack(buf []byte) (Packet, int, error) {
	if len(buf) < 4 {
		return nil, 0, nil
	}

	msgLen := int(binary.BigEndian.Uint32(buf[:4]))
	if len(buf) < msgLen {
		return nil, 0, nil
	}
	msg := &subpub.UniMessage{}
	err := proto.Unmarshal(buf[4:msgLen], msg)
	if err != nil {
		return nil, msgLen, err
	}
	return msg, msgLen, nil
}
