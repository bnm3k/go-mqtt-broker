package protocol

/*
	PING REQUEST PACKET
*/

// PingreqPacket holds the in-memory representation of
// a ping request packet
type PingreqPacket struct{}

// Serialize serializes the contents of a ping request packet into
// a []byte buffer. Buffer should be of appropriate length
// otherwise an error is returned. If nil buffer
// is provided, Serialize instantiates a buffer of required length
// and returns it
func (p *PingreqPacket) Serialize(b []byte) ([]byte, error) {
	if b == nil {
		b = make([]byte, 2)
	}
	if len(b) < 2 { // requires 2 bytes ? or more
		return nil, ErrShortBuffer
	}
	b[0] = Pingreq<<4 | 0x0 // ctrl pkt type + flags(reserved)
	b[1] = 0                // no payload
	return b[:2], nil
}

// Len returns the total length in terms of bytes
// that a ping request packet takes
func (p *PingreqPacket) Len() int {
	// takes up 2 bytes
	return 2
}

/*
	PING RESPONSE PACKET
*/

// PingrespPacket holds the in-memory representation of
// a ping request packet
type PingrespPacket struct{}

// Serialize serializes the contents of a ping response packet into
// a []byte buffer. Buffer should be of appropriate length
// otherwise an error is returned. If nil buffer
// is provided, Serialize instantiates a buffer of required length
// and returns it
func (p *PingrespPacket) Serialize(b []byte) ([]byte, error) {
	if b == nil {
		b = make([]byte, 2)
	}
	if len(b) < 2 { // requires 2 bytes ? or more
		return nil, ErrShortBuffer
	}
	b[0] = Pingresp<<4 | 0x0 // ctrl pkt type + flags(reserved)
	b[1] = 0                 // no payload
	return b[:2], nil
}

// Len returns the total length in terms of bytes
// that a ping response packet takes
func (p *PingrespPacket) Len() int {
	// takes up 2 bytes
	return 2
}
