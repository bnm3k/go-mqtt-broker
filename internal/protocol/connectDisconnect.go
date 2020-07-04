package protocol

import (
	"fmt"
	"io"
)

/*
	CONNECT PACKET
*/

// ConnectPacket holds the in-application deserialization
// of a ConnectPacket
type ConnectPacket struct {
	loginDetailsPresent bool // ok
	willRetain          bool
	willQoS             byte
	willFlag            bool
	cleanSession        bool   // ok
	keepAlive           uint16 // ok
	clientIdentifier    []byte // ok
	willTopic           []byte
	willMessage         []byte
	username            []byte // ok
	password            []byte // ok
}

// ConnectPacketConfig is more of a necessary evil,
// it is used to configure the connect packet during
// instantiation. The alternative was either to  have
// a constructor with lots and lots of parameters or to make
// fields in ConnectPacket public which I tried to avoid
// since the flags too must be made public and that places more
// burden on the end user to make sure the flags set are consistent
// with the fields present.
// The non-primitive ConnectPacketConfig fields should not be modified
// any further after a ConnectPacket is instantiated from the config since
// the constructor shallow copies the fields.
// As a sidenote as to why this is a necessary evil, check out the article
// below, config objects are kind of an antipattern
// https://middlemost.com/object-lifecycle/
type ConnectPacketConfig struct {
	ClientIdentifier, Username, Pass []byte
	KeepAliveSeconds                 uint16
	ShouldCleanSession               bool
	WillTopic, WillMessage           []byte
	WillQoS                          byte
	WillRetain                       bool
}

// NewConnectPacket instantiates a ConnectPacket based on the config object passed.
// To be used by the client rather than the server.
// A nil or zero length cfg.ClientIdentifier indicates that the client intends
// for the broker to assign a unique client identifier for it.
// If the cfg.Username is nil or of len 0, the  username and password flags
// will not be set, plus the respective strings will be of 0 length.
// The Will Flag is set iff both the cfg.WillTopic and cfg.WillMessage are
// of nonzero length. If the WillQoS is invalid, ie not equal to 0x0, 0x1, 0x2
// then an error is returned.
func NewConnectPacket(cfg *ConnectPacketConfig) (*ConnectPacket, error) {
	// validate config
	if cfg.WillQoS > 2 {
		return nil, fmt.Errorf("Invalid QoS %d. Should be 0x0, 0x1 or 0x2", cfg.WillQoS)
	}
	p := new(ConnectPacket)
	// setup credentials
	p.clientIdentifier = cfg.ClientIdentifier
	if len(cfg.Username) > 0 {
		p.loginDetailsPresent = true
		p.username = cfg.Username
		p.password = cfg.Pass
	}
	// setup will
	if len(cfg.WillTopic) > 0 && len(cfg.WillMessage) > 0 {
		p.willFlag = true
		p.willQoS = cfg.WillQoS
		p.willRetain = cfg.WillRetain
		p.willTopic = cfg.WillTopic
		p.willMessage = cfg.WillMessage
	}

	// setup other configurations
	p.keepAlive = cfg.KeepAliveSeconds
	p.cleanSession = cfg.ShouldCleanSession
	return p, nil
}

func (p *ConnectPacket) Read(b []byte) (n int, err error) {
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
	if p.willFlag {
		buf.writeMQTTStr(p.willTopic)
		buf.writeMQTTStr(p.willMessage)
	}
	if p.loginDetailsPresent {
		buf.writeMQTTStr(p.username)
		buf.writeMQTTStr(p.password)
	}
	return buf.bytesWritten(), io.EOF
}

func (p *ConnectPacket) getConnectFlagsByte() byte {
	var b byte = 0
	if p.loginDetailsPresent { // username & password present
		b = b | 0xC0 // 0x80 + 0x40 for both username & pass
	}
	if p.willRetain {
		b = b | 0x20
	}
	b = b | p.willQoS
	if p.willFlag {
		b = b | 0x04
	}
	if p.cleanSession {
		b = b | 0x02
	}
	return b
}

func (p *ConnectPacket) payloadLen() int {
	payloadLen := 10 + // variable Header
		2 + len(p.clientIdentifier) +
		2 + len(p.willTopic) +
		2 + len(p.willMessage) +
		2 + len(p.username) +
		2 + len(p.password)
	return payloadLen
}

// Len returns the total number of bytes the ConnectPacket will take up
func (p *ConnectPacket) Len() int {
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
