package stream

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"reflect"

	"github.com/golang/protobuf/proto"
)

var (
	errUnknownProtobufMsgType = errors.New("Unknown protobuf message type")
	errBufSizeNotEnoughToPack = errors.New("buf size not enough")
	errInvalidPacket          = errors.New("invalid packet")
)

// Protocol type.
type Protocol struct {
}

// PackSize return size need for pack Protobufproto.Message.
func (pro Protocol) PackSize(p proto.Message) int {
	msgName := proto.MessageName(p)
	msgNameLen := len(msgName)
	if msgNameLen == 0 {
		return 0
	}
	return 4 + 4 + msgNameLen + proto.Size(p)
}

// PackTo :
func (pro Protocol) PackTo(pp proto.Message, w io.Writer) (int, error) {
	msgName := proto.MessageName(pp)
	msgNameLen := len(msgName)
	if msgNameLen == 0 {
		return 0, errUnknownProtobufMsgType
	}
	msgLen := 4 + 4 + msgNameLen + proto.Size(pp)

	data, err := proto.Marshal(pp)
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

// Pack :
func (pro Protocol) Pack(p proto.Message) ([]byte, error) {
	len := pro.PackSize(p)
	if len != 0 {
		buf := bytes.NewBuffer(nil)
		_, err := pro.PackTo(p, buf)
		return buf.Bytes(), err
	}
	return nil, errUnknownProtobufMsgType
}

// Unpack :
func (pro Protocol) Unpack(buf []byte) (proto.Message, int, error) {
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
		return nil, msgLen, errUnknownProtobufMsgType
	}
	msg := reflect.New(t.Elem()).Interface().(proto.Message)
	err := proto.Unmarshal(buf[8+msgNameLen:msgLen], msg)
	if err != nil {
		return nil, msgLen, err
	}
	return msg, msgLen, nil
}
