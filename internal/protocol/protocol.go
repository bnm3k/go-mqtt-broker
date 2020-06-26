package protocol

import "io"

func check(condition bool, message string) {
	if condition == false {
		panic("Check failed: " + message)
	}
}

type Reader interface {
	io.Reader
	io.ByteReader
}

func read(r Reader) (err error) {
	_, payloadSize, err := readFixedHeader(r)
	// if no other err, ensure ctrl type and flag correct
	// before proceeding further
	buf := make([]byte, payloadSize)
	_, err = io.ReadFull(r, buf)
	return
}
