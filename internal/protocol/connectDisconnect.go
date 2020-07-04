package protocol

import (
	"io"
)

/*
	CONNECT PACKET
*/
type connectPacket struct {
	protocolLevel       byte
	loginDetailsPresent bool
	willRetain          bool
	willQoS             byte
	willPresent         bool
	cleanSession        bool
	keepAlive           uint16
	clientIdentifier    []byte
	willTopic           []byte
	willMessage         []byte
	username            []byte
	password            []byte
}

func (p *connectPacket) Read(b []byte) (n int, err error) {
	lenConnectPacket := p.Len()
	if len(b) < lenConnectPacket {
		return 0, io.ErrShortBuffer
	}
	if lenConnectPacket > maxPayloadSize {
		return 0, ErrInvalidPacket
	}
	// write fixed header
	buf := newWritableBuf(b)
	buf.WriteByte(connect<<4 | 0x0)
	writePayloadSize(buf, uint32(p.payloadLen()))
	// write protocol name + level version 3.11
	buf.Write([]byte{0, 4, 'M', 'Q', 'T', 'T', 0x04})
	// write connect flags
	buf.WriteByte(p.getConnectFlagsByte())
	// write keep alive (msb then lsb)
	buf.WriteByte(byte(p.keepAlive >> 8))
	buf.WriteByte(byte(p.keepAlive))
	// write payload
	buf.writeMQTTStr(p.clientIdentifier)
	if p.willPresent {
		buf.writeMQTTStr(p.willTopic)
		buf.writeMQTTStr(p.willMessage)
	}
	if p.loginDetailsPresent {
		buf.writeMQTTStr(p.username)
		buf.writeMQTTStr(p.password)
	}
	return buf.bytesWritten(), io.EOF
}

func (p *connectPacket) getConnectFlagsByte() byte {
	var b byte = 0
	if p.loginDetailsPresent { // username & password present
		b = b | 0xC0 // 0x80 + 0x40 for both username & pass
	}
	if p.willRetain {
		b = b | 0x20
	}
	b = b | p.willQoS
	if p.willPresent {
		b = b | 0x04
	}
	if p.cleanSession {
		b = b | 0x02
	}
	return b
}

func (p *connectPacket) payloadLen() int {
	payloadLen := 10 + // variable Header
		2 + len(p.clientIdentifier) +
		2 + len(p.willTopic) +
		2 + len(p.willMessage) +
		2 + len(p.username) +
		2 + len(p.password)
	return payloadLen
}

func (p *connectPacket) Len() int {
	payloadLen := p.payloadLen()
	return 1 + // control pkt type + flags
		lenPayloadSizeField(payloadLen) + // remaining Length field
		payloadLen // variable header + payload
}

/*
	CONNACK PACKET
*/
type connackPacket struct {
	sessionPresent    bool
	connectReturnCode connectReturnCode
}

type connectReturnCode byte

const (
	connAccepted connectReturnCode = iota
	connRefusedUnacceptableProtocol
	connRefusedIdentifierRejected
	connRefusedServerUnavailable
	connRefusedBadUsernamePass
	connRefusedNotAuthorized
)

func (code connectReturnCode) String() string {
	switch code {
	case connAccepted:
		return "Connection accepted"
	case connRefusedUnacceptableProtocol:
		return "The Server does not support the level of the MQTT protocol requested by the Client"
	case connRefusedIdentifierRejected:
		return "The Client identifier is correct UTF-8 but not allowed by the Server"
	case connRefusedServerUnavailable:
		return "The Network Connection has been made but the MQTT service is unavailable"
	case connRefusedBadUsernamePass:
		return "The data in the user name or password is malformed"
	case connRefusedNotAuthorized:
		return "The Client is not authorized to connect"
	default:
		return "Reserved for future use"
	}
}

func (p *connackPacket) Read(b []byte) (n int, err error) {
	if len(b) < 4 { // requires 2 bytes ? or more
		return 0, io.ErrShortBuffer
	}
	b[0] = connack<<4 | 0x0 // ctrl pkt type + flags(reserved)
	b[1] = 2                // remaining length
	if p.sessionPresent {   // session present
		b[2] = 1
	} else {
		b[2] = 0
	}
	b[3] = byte(p.connectReturnCode)
	return 4, io.EOF
}

func (p *connackPacket) Len() int {
	// takes up 4 bytes, 2 for fixed header, 2 for variable header
	return 4
}

func (p *connackPacket) ConnectionAccepted() (ok bool, description string) {
	ok = p.connectReturnCode == connAccepted
	description = p.connectReturnCode.String()
	return
}

/*
	DISCONNECT PACKET
*/
type disconnectPacket struct{}

func (p *disconnectPacket) Read(b []byte) (n int, err error) {
	if len(b) < 2 { // requires 2 bytes ? or more
		return 0, io.ErrShortBuffer
	}
	b[0] = disconnect<<4 | 0x0 // ctrl pkt type + flags(reserved)
	b[1] = 0                   // remaining length, zero
	return 2, io.EOF
}

func (p *disconnectPacket) Len() int {
	// disconnect packets are always 2 bytes
	return 2
}
