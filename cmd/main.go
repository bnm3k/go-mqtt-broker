package main

import (
	"encoding/binary"
	"io"
)

// TLSize is the size of the Type section
// and Length section separately. For example.
// Size of one Byte can accomodate upto 256 types
// and a payload of 256 bytes
type TLSize int

const (
	OneByte    TLSize = 1
	TwoBytes   TLSize = 2
	FourBytes  TLSize = 4
	EightBytes TLSize = 8
)

// Record represents a record of data encoded in the
// TLV message
type Record struct {
	Payload []byte
	Type    uint
}

// Codec is the configuration for a specific TLV
// encoding/decoding tasks.
type Codec struct {
	TypeBytes TLSize
	LenBytes  TLSize
}

// Writer encodse records into TLV format
// using the codex and writes them into a provided io.Writer
type Writer struct {
	writer io.Writer
	codec  *Codec
}

// NewWriter inits and returns an instance of Writer
func NewWriter(w io.Writer, codec *Codec) *Writer {
	return &Writer{
		writer: w,
		codec:  codec,
	}
}

func (w *Writer) Write(rec *Record) error {
	err := writeUint(w.writer, w.codec.TypeBytes, rec.Type)
	if err != nil {
		return err
	}
	ulen := uint(len(rec.Payload))
	err = writeUint(w.writer, w.codec.TypeBytes, ulen)
	if err != nil {
		return err
	}
	_, err = w.writer.Write(rec.Payload)
	return err
}

func writeUint(w io.Writer, b TLSize, i uint) error {
	var num interface{}
	switch b {
	case OneByte:
		num = uint8(i)
	case TwoBytes:
		num = uint16(i)
	case FourBytes:
		num = uint32(i)
	case EightBytes:
		num = uint64(i)
	}
	return binary.Write(w, binary.BigEndian, num)
}

func main() {

}
