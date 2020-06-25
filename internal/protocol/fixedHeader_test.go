package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
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
		require.NoErrorf(t, err, "Unexpected err should be nil %v", err)

		require.Equal(t, cs.expectedBytesWritten, bytesWritten,
			"encode does not write expected number of bytes for %d.\nUsed %d byte(s), expected to use %d byte(s) instead",
			cs.from, bytesWritten, cs.expectedBytesWritten)

		require.Equal(t, cs.to, buf.Bytes(),
			"encode does not encode number(%d) to expected bytes, got %v want %v",
			cs.from, buf.Bytes(), cs.to)

		s := &spyReadPayloadSize{r: buf}
		retrievedPayloadSize, err := readPayloadSize(s)
		require.NoErrorf(t,
			errors.Wrap(err, fmt.Sprintf("num to be encoded(%d), bytes count(%d)", cs.from, bytesWritten)),
			"error should be nil since encoded payload size is valid.")

		require.Equal(t, cs.from, retrievedPayloadSize,
			"decodePayloadSize(encode(n))!= n. Got %d, want %d",
			retrievedPayloadSize, cs.from)

		require.Equal(t, bytesWritten, s.numberOfBytesReadSoFar,
			"number of bytes read so far is more/less than expected. Got %d, want %d",
			s.numberOfBytesReadSoFar, bytesWritten)

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
		require.Error(t, err,
			"payload size is invalid, should return an error")
		require.EqualError(t, err, errMalformedRemainingSize.Error(),
			"error should be of value errMalformedRemainingSize, instead is: %v", err)
	}
}
