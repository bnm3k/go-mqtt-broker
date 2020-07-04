package protocol

import "io"

/*
	PING REQUEST PACKET
*/
type pingreqPacket struct{}

func (p *pingreqPacket) Read(buf []byte) (n int, err error) {
	if len(buf) < 2 { // requires 2 bytes ? or more
		return 0, io.ErrShortBuffer
	}
	buf[0] = pingreq<<4 | 0x0 // ctrl pkt type + flags(reserved)
	buf[1] = 0                // no payload
	return 2, io.EOF
}

func (p *pingreqPacket) Len() int {
	// takes up 2 bytes
	return 2
}

/*
	PING RESPONSE PACKET
*/
type pingrespPacket struct{}

func (p *pingrespPacket) Read(buf []byte) (n int, err error) {
	if len(buf) < 2 { // requires 2 bytes ? or more
		return 0, io.ErrShortBuffer
	}
	buf[0] = pingresp<<4 | 0x0 // ctrl pkt type + flags(reserved)
	buf[1] = 0                 // no payload
	return 2, io.EOF
}

func (p *pingrespPacket) Len() int {
	// takes up 2 bytes
	return 2
}
