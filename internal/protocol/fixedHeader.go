package protocol

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

var maxPayloadSize int = 268435455

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
