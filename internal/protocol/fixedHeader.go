package protocol

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

const maxPayloadSize = 268435455

// FixedHeader represents the content of a fixed header for
// a given mqtt packet
type FixedHeader struct {
	PayloadSize uint32
	PktType     byte
	CtrlFlags   byte
}

// IsValidFlagsSet checks if the correct flags are set
func (f FixedHeader) IsValidFlagsSet() bool {
	// check section 2.2.2 on the default flags to be set
	switch f.PktType {
	case Publish:
		// check section 3.3.1.2 on QoS
		// from spec: A PUBLISH Packet MUST NOT have both QoS
		// bits set to 1. If a Server or Client receives a PUBLISH
		// Packet which has both QoS bits set to 1 it MUST close
		// the Network Connection
		return f.CtrlFlags&0x06 != 0x06
	case Pubrel, Subscribe, Unsubscribe:
		return f.CtrlFlags == 0x02
	default:
		return f.CtrlFlags == 0x00
	}
}

func lenPayloadSizeField(n int) int {
	bytesToWrite := 0
	for {
		n = n >> 7
		bytesToWrite++
		if n == 0 {
			break
		}
	}
	return bytesToWrite
}

func writePayloadSize(w io.ByteWriter, n uint32) (bytesWritten int, err error) {
	for {
		encodedByte := byte(n) % 0x80
		n = n >> 7
		if n > 0 {
			encodedByte = encodedByte | 0x80
		}
		err = w.WriteByte(encodedByte)
		bytesWritten++
		if n == 0 {
			break
		}
	}
	return
}

func writeFixedHeader(w io.ByteWriter, ctrl byte, payloadSize uint32) (bytesWritten int, err error) {
	err = w.WriteByte(ctrl)
	if err != nil {
		return
	}
	bytesWritten, err = writePayloadSize(w, payloadSize)
	bytesWritten++
	return
}

var errMalformedRemainingSize error = fmt.Errorf("Malformed remaining size. 4th byte's 8th bit indicates continue")

// if last byte read indicates that more bytes should be read
// but the io.ByteReader returns an error such as io.EOF,
// the function returns an error
func readPayloadSize(r io.ByteReader) (val uint32, err error) {
	// read first byte
	encodedByte, err := r.ReadByte()
	val = uint32(encodedByte & 0x7F)
	if err != nil {
		return val, errors.Wrap(err, "Error reading byte when decoding payload size")
	}
	// read rest of bytes
	var bytesRead int = 1
	for {
		// check whether 8th bit indicates continuation
		// if 1 continue, if 0 break
		if (encodedByte & 0x80) == 0 {
			break
		}
		// if bytesRead is already 4 and the 4th encodedByte
		// indicates that we should continue reading more bytes
		// OR the encoded byte indicates we should read more bytes
		// but there aren't any more bytes to read
		// this is an error as per the specification, hence stop
		if bytesRead == 4 {
			return val, errMalformedRemainingSize
		}
		// otherwise, proceed as usual
		encodedByte, err = r.ReadByte()
		if err == io.EOF {
			return val, errMalformedRemainingSize
		} else if err != nil {
			return val, errors.Wrap(err, "Error reading byte when decoding payload size")
		}
		val += uint32(encodedByte&0x7F) << (7 * bytesRead)
		bytesRead++
	}
	return val, nil
}

// ReadFixedHeader2 retrieves both the ctrl byte(type + flags) and payloadSize from
// an io.ByteReader such as net.Conn
func ReadFixedHeader2(r io.ByteReader) (pktType, flags byte, payloadSize uint32, err error) {
	ctrl, err := r.ReadByte()
	if err != nil {
		return
	}
	pktType = ctrl >> 4
	flags = ctrl & 0x0F
	payloadSize, err = readPayloadSize(r)
	return
}

// ReadFixedHeader retrieves both the ctrl byte(type + flags) and payloadSize from
// an io.ByteReader such as net.Conn
func ReadFixedHeader(r io.ByteReader) (f FixedHeader, err error) {
	ctrl, err := r.ReadByte()
	if err != nil {
		return
	}
	f.PktType = ctrl >> 4
	f.CtrlFlags = ctrl & 0x0F
	f.PayloadSize, err = readPayloadSize(r)
	return
}
