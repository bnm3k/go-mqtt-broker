package protocol

import "io"

/*
	PING REQUEST PACKET
*/
type pingreqPacket struct{}

func (p *pingreqPacket) Read(b []byte) (n int, err error) {
	if len(b) < 2 { // requires 2 bytes ? or more
		return 0, io.ErrShortBuffer
	}
	b[0] = pingreq<<4 | 0x0 // ctrl pkt type + flags(reserved)
	b[1] = 0                // no payload
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

func (p *pingrespPacket) Read(b []byte) (n int, err error) {
	if len(b) < 2 { // requires 2 bytes ? or more
		return 0, io.ErrShortBuffer
	}
	b[0] = pingresp<<4 | 0x0 // ctrl pkt type + flags(reserved)
	b[1] = 0                 // no payload
	return 2, io.EOF
}

func (p *pingrespPacket) Len() int {
	// takes up 2 bytes
	return 2
}
