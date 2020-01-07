package stream

type Statistics struct {
	ByteSend      uint64
	ByteRecv      uint64
	PacketSend    uint64
	PacketSendErr uint64
	PacketRecv    uint64
	PacketRecvErr uint64
}

func AddStat(a, b Statistics) Statistics {
	return Statistics{
		ByteSend:      a.ByteSend + b.ByteSend,
		ByteRecv:      a.ByteRecv + b.ByteRecv,
		PacketSend:    a.PacketSend + b.PacketSend,
		PacketSendErr: a.PacketSendErr + b.PacketSendErr,
		PacketRecv:    a.PacketRecv + b.PacketRecv,
		PacketRecvErr: a.PacketRecvErr + b.PacketRecvErr,
	}
}

func SubStat(a, b Statistics) Statistics {
	return Statistics{
		ByteSend:      a.ByteSend - b.ByteSend,
		ByteRecv:      a.ByteRecv - b.ByteRecv,
		PacketSend:    a.PacketSend - b.PacketSend,
		PacketSendErr: a.PacketSendErr - b.PacketSendErr,
		PacketRecv:    a.PacketRecv - b.PacketRecv,
		PacketRecvErr: a.PacketRecvErr - b.PacketRecvErr,
	}
}

func StatPerSec(a Statistics, sec float64) Statistics {
	return Statistics{
		ByteSend:      uint64(float64(a.ByteSend) / sec),
		ByteRecv:      uint64(float64(a.ByteRecv) / sec),
		PacketSend:    uint64(float64(a.PacketSend) / sec),
		PacketSendErr: uint64(float64(a.PacketSendErr) / sec),
		PacketRecv:    uint64(float64(a.PacketRecv) / sec),
		PacketRecvErr: uint64(float64(a.PacketRecvErr) / sec),
	}
}
