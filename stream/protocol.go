package stream

import (
	"fmt"
	"io"
)

// EventHandler ...
type EventHandler interface {
	// OnConnAccepted means connection is accepted.
	OnConnAccepted(c *Conn)
	// OnPackReceived can use c to send reply message.
	OnPackReceived(c *Conn, pack Packet)
	// OnConnClosed means connection is closed.
	OnConnClosed(c *Conn)
}

// OnPackReceived can use c to send reply message
type OnPackReceived func(c *Conn, pack Packet)

// todo we may choose another underlying serialization tool, for example, cap'n.
// so the parameter is not necessarily proto.Message.
type Packet interface{}

var (
	errUnknownMsgType = fmt.Errorf("unknown message type")
	errInvalidPacket  = fmt.Errorf("invalid packet")
)

// Protocol can be any protocol we want
type Protocol interface {
	// PackSize return size need for pack Protobuf proto.Message.
	PackSize(pack Packet) int
	// PackInto a io.Writer
	PackInto(pack Packet, w io.Writer) (int, error)
	// Unpack from a buf
	Unpack(buf []byte) (Packet, int, error)
}
