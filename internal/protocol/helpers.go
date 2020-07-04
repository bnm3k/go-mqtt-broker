package protocol

import "errors"

// ErrInvalidPacket ...
var ErrInvalidPacket = errors.New("Packet content is invalid")

// Reader ...
// type Reader interface {
// 	io.Reader
// 	io.ByteReader
// }

// func read(r Reader) (err error) {
// 	_, payloadSize, err := readFixedHeader(r)
// 	// if no other err, ensure ctrl type and flag correct
// 	// before proceeding further
// 	buf := make([]byte, payloadSize)
// 	_, err = io.ReadFull(r, buf)
// 	return
// }

func check(condition bool, message string) {
	if condition == false {
		panic("Check failed: " + message)
	}
}

type writableBuf struct {
	buf            []byte
	lastWriteIndex int
}

func newWritableBuf(b []byte) writableBuf {
	return writableBuf{
		buf:            b,
		lastWriteIndex: -1,
	}
}

func (b writableBuf) bytesWritten() int {
	return b.lastWriteIndex + 1
}

func (b writableBuf) WriteByte(c byte) (err error) {
	b.lastWriteIndex++
	b.buf[b.lastWriteIndex] = c
	return
}

func (b writableBuf) Write(p []byte) (n int, err error) {
	n = len(p)
	copy(b.buf[b.lastWriteIndex+1:], p)
	b.lastWriteIndex += n
	return
}

func (b writableBuf) writeMQTTStr(str []byte) {
	var strLen uint16 = uint16(len(str))
	b.WriteByte(byte(strLen >> 8))
	b.WriteByte(byte(strLen))
	b.Write(str[:strLen])
}
