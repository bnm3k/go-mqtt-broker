package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"testing"

	"github.com/pkg/errors"
)

// spyPayloadSizeDecoder used to 'spy' on decode fn
// to make sure that it reads only the necessary number
// of bytes to return the payload size
type spyReadPayloadSize struct {
	r                      io.ByteReader
	numberOfBytesReadSoFar int
}

func (s *spyReadPayloadSize) ReadByte() (byte, error) {
	b, err := s.r.ReadByte()
	if err == nil {
		s.numberOfBytesReadSoFar++
	}
	return b, err
}

func TestReadWritePayloadSize(t *testing.T) {
	cases := []struct {
		from                 uint32
		to                   []byte
		expectedBytesWritten int
	}{
		{0, []byte{0x00}, 1},
		{127, []byte{0x7F}, 1},
		{128, []byte{0x80, 0x01}, 2},
		{16383, []byte{0xFF, 0x7F}, 2},
		{16384, []byte{0x80, 0x80, 0x01}, 3},
		{2097151, []byte{0xFF, 0xFF, 0x7F}, 3},
		{2097152, []byte{0x80, 0x80, 0x80, 0x01}, 4},
		{268435455, []byte{0xFF, 0xFF, 0xFF, 0x7F}, 4},
	}
	buf := bytes.NewBuffer(make([]byte, binary.MaxVarintLen32))
	for _, cs := range cases {
		buf.Reset()
		bytesWritten, err := writePayloadSize(buf, cs.from)
		if err != nil {
			t.Errorf("Unexpected err should be non-nil %v", err)
		}
		if bytesWritten != cs.expectedBytesWritten {
			t.Errorf("encode does not write expected number of bytes for %d.\nUsed %d byte(s), expected to use %d byte(s) instead", cs.from, bytesWritten, cs.expectedBytesWritten)
		}
		if !bytes.Equal(buf.Bytes(), cs.to) {
			t.Errorf("encode does not encode number(%d) to expected bytes, got %v want %v", cs.from, buf, cs.to)
		}

		s := &spyReadPayloadSize{r: buf}
		retrievedPayloadSize, err := readPayloadSize(s)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("num to be encoded(%d), bytes count(%d)", cs.from, bytesWritten))
			t.Errorf("error should be nil since encoded payload size is valid.\n Err: %s", err)
		}
		if retrievedPayloadSize != cs.from {
			t.Errorf("decodePayloadSize(encode(n))!= n. Got %d, want %d", retrievedPayloadSize, cs.from)
		}
		if s.numberOfBytesReadSoFar != bytesWritten {
			t.Errorf("number of bytes read so far is more/less than expected. Got %d, want %d", s.numberOfBytesReadSoFar, bytesWritten)
		}
	}
}

func TestReadInvalidPayloadSize(t *testing.T) {
	cases := []struct {
		encoded []byte
	}{
		{[]byte{0x80}},
		{[]byte{0x80, 0x80}},
		{[]byte{0x80, 0x80, 0x80}},
		{[]byte{0x80, 0x80, 0x80, 0x80}},
	}
	for _, cs := range cases {
		_, err := readPayloadSize(bytes.NewBuffer(cs.encoded))
		if err != errMalformedRemainingSize {
			t.Errorf("error should be of value errMalformedRemainingSize, instead is %v", err)
		}
	}
}
