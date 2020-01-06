package stream

import (
	"encoding/binary"
	"io"
	"reflect"

	"github.com/golang/protobuf/proto"
)

// Protobuf is a universal protobuf protocol
type Protobuf struct{}

func (p Protobuf) PackSize(pack Packet) int {
	pb, ok := pack.(proto.Message)
	if !ok {
		return 0
	}
	msgName := proto.MessageName(pb)
	msgNameLen := len(msgName)
	if msgNameLen == 0 {
		return 0
	}
	return 4 + 4 + msgNameLen + proto.Size(pb)
}

func (p Protobuf) PackInto(pack Packet, w io.Writer) (int, error) {
	pb, ok := pack.(proto.Message)
	if !ok {
		return 0, errUnknownMsgType
	}
	msgName := proto.MessageName(pb)
	msgNameLen := len(msgName)
	if msgNameLen == 0 {
		return 0, errUnknownMsgType
	}
	msgLen := 4 + 4 + msgNameLen + proto.Size(pb)

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
	err = binary.Write(w, binary.BigEndian, uint32(msgNameLen))
	if err != nil {
		return wl, err
	}
	wl += 4
	n, err := w.Write([]byte(msgName))
	wl += n
	if err != nil {
		return wl, err
	}
	n, err = w.Write(data)
	wl += n
	if err != nil {
		return wl, err
	}
	return wl, nil
}

func (p Protobuf) Unpack(buf []byte) (Packet, int, error) {
	if len(buf) < 4 {
		return nil, 0, nil
	}

	msgLen := int(binary.BigEndian.Uint32(buf[:4]))
	if len(buf) < msgLen {
		return nil, 0, nil
	}

	msgNameLen := binary.BigEndian.Uint32(buf[4:8])
	if int(msgNameLen)+8 > msgLen {
		return nil, msgLen, errInvalidPacket
	}
	msgName := string(buf[8 : 8+msgNameLen])

	t := proto.MessageType(msgName)
	if t == nil {
		return nil, msgLen, errUnknownMsgType
	}
	msg := reflect.New(t.Elem()).Interface().(proto.Message)
	err := proto.Unmarshal(buf[8+msgNameLen:msgLen], msg)
	if err != nil {
		return nil, msgLen, err
	}
	return msg, msgLen, nil
}
