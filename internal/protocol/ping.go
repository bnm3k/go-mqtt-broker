package protocol

/*
	PING REQUEST PACKET
*/
type PingreqPacket struct{}

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

func (p *PingreqPacket) Len() int {
	// takes up 2 bytes
	return 2
}

/*
	PING RESPONSE PACKET
*/
type PingrespPacket struct{}

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

func (p *PingrespPacket) Len() int {
	// takes up 2 bytes
	return 2
}
