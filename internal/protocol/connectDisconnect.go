package protocol

import (
	"bytes"
	"fmt"
)

/*
	CONNECT PACKET
*/

// ConnectPacket holds the in-application deserialization
// of a ConnectPacket
type ConnectPacket struct {
	usernamePresent  bool
	passwordPresent  bool
	willRetain       bool
	willQoS          byte
	willFlag         bool
	cleanSession     bool   // ok
	keepAlive        uint16 // ok
	clientIdentifier []byte // ok
	willTopic        []byte
	willMessage      []byte
	username         []byte // ok
	password         []byte // ok
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
	ClientIdentifier, Username, Password []byte
	KeepAliveSeconds                     uint16
	ShouldCleanSession                   bool
	WillTopic, WillMessage               []byte
	WillQoS                              byte
	WillRetain                           bool
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
	// setup username & password
	p.clientIdentifier = cfg.ClientIdentifier
	if len(cfg.Username) > 0 {
		p.usernamePresent = true
		p.username = cfg.Username
		if len(cfg.Password) > 0 {
			p.passwordPresent = true
			p.password = cfg.Password
		}
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

var protocolVersion []byte = []byte{0, 4, 'M', 'Q', 'T', 'T', 0x04}

// Serialize serializes the contents of a connect packet into
// a []byte buffer. Buffer should be of appropriate length
// otherwise a ErrShortBuffer error is returned. If nil buffer
// is provided, Serialize instantiates a buffer of required length
// and returns it
func (p *ConnectPacket) Serialize(b []byte) ([]byte, error) {
	lenConnectPacket := p.Len()
	if b == nil {
		b = make([]byte, lenConnectPacket)
	}
	if len(b) < lenConnectPacket {
		return nil, ErrShortBuffer
	}
	if lenConnectPacket > maxPayloadSize {
		return nil, ErrInvalidPacket
	}
	// write fixed header
	buf := newWritableBuf(b)
	buf.WriteByte(Connect<<4 | 0x0)
	writePayloadSize(buf, uint32(p.payloadLen()))
	// write protocol name + level version 3.11
	buf.Write(protocolVersion)
	// write connect flags
	flags := p.getConnectFlagsByte()
	buf.WriteByte(flags)
	// write keep alive (msb then lsb)
	buf.WriteByte(byte(p.keepAlive >> 8))
	buf.WriteByte(byte(p.keepAlive))
	// write payload
	buf.writeMQTTStr(p.clientIdentifier)
	if p.willFlag {
		buf.writeMQTTStr(p.willTopic)
		buf.writeMQTTStr(p.willMessage)
	}
	if p.usernamePresent {
		buf.writeMQTTStr(p.username)
		if p.passwordPresent {
			buf.writeMQTTStr(p.password)
		}
	}

	return b[:buf.bytesWritten()], nil
}

// DeserializeConnectPktPayload parses the contents of a bytes slice and returns
// a ConnectPacket as required.
func DeserializeConnectPktPayload(ctrlFlags byte, p []byte) (*ConnectPacket, error) {
	var err error
	readStr := func(from []byte, startPos int) (str []byte, nextPos int) {
		if err == nil {
			from = from[startPos:]
			if len(from) < 2 {
				err = ErrInvalidPacket
				return
			}
			strLen := (int(from[0]) << 8) + int(from[1])
			if len(from) < strLen+2 {
				err = ErrInvalidPacket
				return
			}
			if strLen > 0 {
				str = from[2 : 2+strLen]
			}
			nextPos = startPos + 2 + strLen
		}
		return
	}
	// check valid ctrl flags set, ie reserved
	if ctrlFlags != 0x00 {
		return nil, ErrInvalidPacket
	}
	// payload must be at least 12 bytes to be valid
	if len(p) < 12 {
		return nil, ErrInvalidPacket
	}
	// protocol version must be valid
	if !bytes.Equal(p[:7], protocolVersion) {
		return nil, ErrInvalidPacket
	}
	flags := p[7]
	// reserved flag bit should not be set
	if flags&0x01 != 0 {
		return nil, ErrInvalidPacket
	}
	// password flag bit set iff username flag bit set
	usernamePresent, passwordPresent := (flags&0x80) == 0x80, (flags&0x40) == 0x40
	if !usernamePresent && passwordPresent {
		return nil, ErrInvalidPacket
	}
	// qos must be 0, 1 or 2
	willQoS := (flags >> 3) & 0x03
	if willQoS > 2 {
		return nil, ErrInvalidPacket
	}

	// if willFlag set to 0, willQoS and willRetain must be zero
	willFlag := (flags & 0x04) == 0x04
	willRetain := (flags & 0x20) == 0x20
	if willFlag == false && willQoS != 0 && willRetain == true {
		return nil, ErrInvalidPacket
	}

	pkt := &ConnectPacket{
		usernamePresent: usernamePresent,
		passwordPresent: passwordPresent,
		willRetain:      willRetain,
		willQoS:         willQoS,
		willFlag:        willFlag,
		cleanSession:    (flags & 0x02) == 0x02,
		keepAlive:       (uint16(p[8]) << 8) + uint16(p[9]),
	}

	// get client identifier
	i := 10
	pkt.clientIdentifier, i = readStr(p, i)

	// if client sets cleanSession to false but does not
	// provde a client ID, packet is invalid
	if pkt.clientIdentifier == nil && !pkt.cleanSession {
		return nil, ErrInvalidPacket
	}
	// get will flag & will message
	if pkt.willFlag {
		pkt.willTopic, i = readStr(p, i)
		pkt.willMessage, i = readStr(p, i)
	}
	// get username
	if pkt.usernamePresent {
		pkt.username, i = readStr(p, i)
	}
	// get password
	if pkt.passwordPresent {
		pkt.password, i = readStr(p, i)
	}

	return pkt, err
}

func (p *ConnectPacket) getConnectFlagsByte() byte {
	var b byte = 0
	if p.usernamePresent { // username & password present
		b = b | 0x80
		if p.passwordPresent {
			b = b | 0x40
		}
	}
	if p.willRetain {
		b = b | 0x20
	}
	b = b | (p.willQoS << 3)
	if p.willFlag {
		b = b | 0x04
	}
	if p.cleanSession {
		b = b | 0x02
	}
	return b
}

// PayloadLen returns length of payload, ie minus fixed header size
func (p *ConnectPacket) payloadLen() int {
	payloadLen := 10 + // variable Header
		2 + len(p.clientIdentifier)
	if p.willFlag {
		payloadLen += 2 + len(p.willTopic) + 2 + len(p.willMessage)
	}
	if p.usernamePresent {
		payloadLen += 2 + len(p.username)
	}
	if p.passwordPresent {
		payloadLen += 2 + len(p.password)
	}
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

// ConnackPacket holds the in-memory representation of
// a connack packet
type ConnackPacket struct {
	SessionPresent    bool
	ConnectReturnCode ConnectReturnCode
}

// ConnectReturnCode holds the return code for
// when a broker responds to a client's connect
// packet
type ConnectReturnCode byte

// ConnAccepted etc self-explanatory
const (
	ConnAccepted ConnectReturnCode = iota
	ConnRefusedUnacceptableProtocol
	ConnRefusedIdentifierRejected
	ConnRefusedServerUnavailable
	ConnRefusedBadUsernamePass
	ConnRefusedNotAuthorized
)

func (code ConnectReturnCode) String() string {
	switch code {
	case ConnAccepted:
		return "Connection accepted"
	case ConnRefusedUnacceptableProtocol:
		return "The Server does not support the level of the MQTT protocol requested by the Client"
	case ConnRefusedIdentifierRejected:
		return "The Client identifier is correct UTF-8 but not allowed by the Server"
	case ConnRefusedServerUnavailable:
		return "The Network Connection has been made but the MQTT service is unavailable"
	case ConnRefusedBadUsernamePass:
		return "The data in the user name or password is malformed"
	case ConnRefusedNotAuthorized:
		return "The Client is not authorized to connect"
	default:
		return "Reserved for future use"
	}
}

// DeserializeConnackPktPayload parses the contents of a bytes slice and returns
// a ConnackPacket as required.
func DeserializeConnackPktPayload(ctrlFlags byte, p []byte) (*ConnackPacket, error) {
	// check control flags are valid (reserved values)
	if ctrlFlags != 0x00 {
		return nil, ErrInvalidPacket
	}
	// payload must be of length 2
	if len(p) < 2 {
		return nil, ErrInvalidPacket
	}
	connAckFlags := p[0]
	// first 7 bits of connect ack flags must be 0
	if (connAckFlags & 0xFE) != 0xFE {
		return nil, ErrInvalidPacket
	}
	// connect return code should not use reserved values
	cr := p[1]
	if cr > 5 {
		return nil, ErrInvalidPacket
	}
	pkt := &ConnackPacket{
		SessionPresent:    (connAckFlags & 0x01) == 0x01,
		ConnectReturnCode: ConnectReturnCode(cr),
	}
	return pkt, nil
}

// Serialize serializes the contents of a connack packet into
// a []byte buffer. Buffer should be of appropriate length
// otherwise a ErrShortBuffer error is returned. If nil buffer
// is provided, Serialize instantiates a buffer of required length
// and returns it
func (p *ConnackPacket) Serialize(b []byte) ([]byte, error) {
	if b == nil {
		b = make([]byte, 4)
	}
	if len(b) < 4 { // requires 2 bytes ? or more
		return nil, ErrShortBuffer
	}
	b[0] = Connack<<4 | 0x0 // ctrl pkt type + flags(reserved)
	b[1] = 2                // remaining length
	if p.SessionPresent {   // session present
		b[2] = 1
	} else {
		b[2] = 0
	}
	b[3] = byte(p.ConnectReturnCode)
	return b[:4], nil
}

// Len returns the total length in terms of bytes
// that the Connack packet takes
func (p *ConnackPacket) Len() int {
	// takes up 4 bytes, 2 for fixed header, 2 for variable header
	return 4
}

// ConnectionAccepted is a convenience method that allows the user, ie a
// client to check whether their connection was accepted and if not, the
// reason why
func (p *ConnackPacket) ConnectionAccepted() (ok bool, description string) {
	ok = p.ConnectReturnCode == ConnAccepted
	description = p.ConnectReturnCode.String()
	return
}

/*
	DISCONNECT PACKET
*/

// DisconnectPacket holds the in-memory representation of a disconnect packet
type DisconnectPacket struct{}

// Serialize serializes the contents of a disconnect packet into
// a []byte buffer. Buffer should be of appropriate length
// otherwise a ErrShortBuffer error is returned. If nil buffer
// is provided, Serialize instantiates a buffer of required length
// and returns it
func (p *DisconnectPacket) Serialize(b []byte) ([]byte, error) {
	if b == nil {
		b = make([]byte, 4)
	}
	if len(b) < 2 { // requires 2 bytes ? or more
		return nil, ErrShortBuffer
	}
	b[0] = Disconnect<<4 | 0x0 // ctrl pkt type + flags(reserved)
	b[1] = 0                   // remaining length, zero
	return b[:2], nil
}

// Len returns the total length in terms of bytes
// that the disconnect packet takes
func (p *DisconnectPacket) Len() int {
	// disconnect packets are always 2 bytes
	return 2
}
