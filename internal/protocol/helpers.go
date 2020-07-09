package protocol

import "errors"

// ErrInvalidPacket ...
var ErrInvalidPacket = errors.New("Packet content is invalid")

// ErrShortBuffer ...
var ErrShortBuffer = errors.New("Buffer is too short")

func check(condition bool, message string) {
	if condition == false {
		panic("Check failed: " + message)
	}
}

type writableBuf struct {
	buf            []byte
	lastWriteIndex int
}

func newWritableBuf(b []byte) *writableBuf {
	return &writableBuf{
		buf:            b,
		lastWriteIndex: -1,
	}
}

func (b *writableBuf) bytesWritten() int {
	return b.lastWriteIndex + 1
}

func (b *writableBuf) WriteByte(c byte) (err error) {
	b.lastWriteIndex++
	b.buf[b.lastWriteIndex] = c
	return
}

func (b *writableBuf) WriteUInt16(n uint16) (err error) {
	b.buf[b.lastWriteIndex+1] = byte(n >> 8)
	b.buf[b.lastWriteIndex+2] = byte(n)
	b.lastWriteIndex += 2
	return
}

func (b *writableBuf) Write(p []byte) (n int, err error) {
	n = len(p)
	copy(b.buf[b.lastWriteIndex+1:], p)
	b.lastWriteIndex += n
	return
}

func (b *writableBuf) writeMQTTStr(str []byte) {
	var strLen uint16 = uint16(len(str))
	b.WriteByte(byte(strLen >> 8))
	b.WriteByte(byte(strLen))
	b.Write(str[:strLen])
}

type pktReader struct {
	i    int
	err  error
	from []byte
}

func (r *pktReader) readUInt16() (n uint16) {
	if r.err == nil {
		if r.i+2 > len(r.from) {
			r.err = ErrInvalidPacket
			return
		}
		n = (uint16(r.from[r.i]) << 8) + uint16(r.from[r.i+1])
		r.i += 2
	}
	return
}

func (r *pktReader) readByte() (b byte) {
	if r.err == nil {
		if r.i+1 > len(r.from) {
			r.err = ErrInvalidPacket
			return
		}
		b = r.from[r.i]
		r.i++
	}
	return
}

func (r *pktReader) isReadComplete() bool {
	return r.i >= len(r.from)
}

func (r *pktReader) readStr() (str []byte) {
	if r.err == nil {
		if r.i+2 > len(r.from) {
			r.err = ErrInvalidPacket
			return
		}
		strLen := (int(r.from[r.i]) << 8) + int(r.from[r.i+1])
		r.i += 2

		if r.i+strLen > len(r.from) {
			r.err = ErrInvalidPacket
			return
		}
		if strLen > 0 {
			str = r.from[r.i : r.i+strLen]
			r.i += strLen
		}
	}
	return
}

func (r *pktReader) readBuf(bufLen int) (buf []byte) {
	if r.err == nil {
		if r.i > len(r.from) {
			r.err = ErrInvalidPacket
			return
		}
		if r.i < len(r.from) {
			buf = r.from[r.i : r.i+bufLen]
		}
	}
	return
}
