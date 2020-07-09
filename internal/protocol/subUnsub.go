package protocol

import "fmt"

// TopicQoS holds both a topic and it's qos
type TopicQoS struct {
	Topic []byte
	Qos   byte
}

// SubscribePacket is an in-mem representation
// of a sub packet
type SubscribePacket struct {
	PacketIdentifier uint16
	List             []TopicQoS
}

// AddTopic adds a given topic plus QoS level to the Subscribe Packet,
// It checks if the topic and qos levels are valid, if not, returns an error
func (p *SubscribePacket) AddTopic(topic []byte, QoS byte) error {
	if QoS > 2 {
		return fmt.Errorf("Invalid QoS: %d", QoS)
	}
	// TODO add error checking for topics
	p.List = append(p.List, TopicQoS{topic, QoS})
	return nil
}

// Serialize serializes the contents of a subscribe packet into
// a []byte buffer.
func (p *SubscribePacket) Serialize(b []byte) ([]byte, error) {
	lenPkt := p.Len()
	if b == nil {
		b = make([]byte, lenPkt)
	}
	if len(b) < lenPkt {
		return nil, ErrShortBuffer
	}
	if lenPkt > maxPayloadSize {
		return nil, ErrInvalidPacket
	}
	// write fixed header
	buf := newWritableBuf(b)
	buf.WriteByte(Subscribe<<4 | 0x02)
	writePayloadSize(buf, uint32(p.payloadLen()))
	// write packet identifier
	buf.WriteUInt16(p.PacketIdentifier)
	// write topics
	for _, t := range p.List {
		buf.writeMQTTStr(t.Topic)
		buf.WriteByte(t.Qos)
	}

	return b[:buf.bytesWritten()], nil
}

func (p *SubscribePacket) payloadLen() int {
	payloadLen := 2 // for packet identifier
	for _, t := range p.List {
		payloadLen = payloadLen + 2 + len(t.Topic) + 1
	}
	return payloadLen
}

// Len returns number of bytes subscribe packet will
// take when serialized
func (p *SubscribePacket) Len() int {
	payloadLen := p.payloadLen()
	return 1 + // control pkt type + flags
		lenPayloadSizeField(payloadLen) + // remaining length field
		payloadLen
}

// DeserializeSubscribePktPayload parses the contents of a bytes slice and returns
// a PublishPacket as required.
func DeserializeSubscribePktPayload(f FixedHeader, p []byte) (*SubscribePacket, error) {
	if !f.IsValidFlagsSet() {
		return nil, ErrInvalidPacket
	}

	pkt := &SubscribePacket{}
	pr := &pktReader{from: p}
	pkt.PacketIdentifier = pr.readUInt16()
	for !pr.isReadComplete() {
		topic := pr.readStr()
		qos := pr.readByte()
		if err := pkt.AddTopic(topic, qos); err != nil {
			return nil, err
		}
	}
	if pr.err != nil {
		return nil, pr.err
	}
	return pkt, nil
}

// SubackPacket is an in-mem representation
// of a sub packet
type SubackPacket struct {
	PacketIdentifier uint16
	ReturnCodes      []byte
}

// AddQoSGranted adds the QoS granted
func (p *SubackPacket) AddQoSGranted(qos byte) error {
	if qos > 2 {
		return fmt.Errorf("invalid qos code: %d", qos)
	}
	p.ReturnCodes = append(p.ReturnCodes, qos)
	return nil
}

// AddFailure adds a failure code
func (p *SubackPacket) AddFailure() {
	p.ReturnCodes = append(p.ReturnCodes, 0x80)
}

// AddCode might be failure or QoS
func (p *SubackPacket) AddCode(c byte) error {
	if c > 2 && c != 0x80 {
		return fmt.Errorf("invalid code: %d", c)
	}
	p.ReturnCodes = append(p.ReturnCodes, c)
	return nil
}

// Serialize serializes the contents of a suback packet into
// a []byte buffer.
func (p *SubackPacket) Serialize(b []byte) ([]byte, error) {
	lenPkt := p.Len()
	if b == nil {
		b = make([]byte, lenPkt)
	}
	if len(b) < lenPkt {
		return nil, ErrShortBuffer
	}
	if lenPkt > maxPayloadSize {
		return nil, ErrInvalidPacket
	}

	// write fixed header
	buf := newWritableBuf(b)
	buf.WriteByte(Suback<<4 | 0x00)
	writePayloadSize(buf, uint32(p.payloadLen()))
	// write packet identifier
	buf.WriteUInt16(p.PacketIdentifier)
	// write return codes
	for _, c := range p.ReturnCodes {
		buf.WriteByte(c)
	}

	return b[:buf.bytesWritten()], nil
}

func (p *SubackPacket) payloadLen() int {
	return 2 + len(p.ReturnCodes)
}

// Len returns number of bytes suback packet will
// take when serialized
func (p *SubackPacket) Len() int {
	payloadLen := p.payloadLen()
	return 1 + // control pkt type + flags
		lenPayloadSizeField(payloadLen) + // remaining length field
		payloadLen
}

// DeserializeSubackPktPayload parses the contents of a bytes slice and returns
// a PublishPacket as required.
func DeserializeSubackPktPayload(f FixedHeader, p []byte) (*SubackPacket, error) {
	if !f.IsValidFlagsSet() {
		return nil, ErrInvalidPacket
	}

	pkt := &SubackPacket{}
	pr := &pktReader{from: p}
	pkt.PacketIdentifier = pr.readUInt16()
	for !pr.isReadComplete() {
		c := pr.readByte()
		if err := pkt.AddCode(c); err != nil {
			return nil, err
		}
	}
	if pr.err != nil {
		return nil, pr.err
	}
	return pkt, nil
}

// UnsubscribePacket is an in-mem representation
// of a Unsubscribe packet
type UnsubscribePacket struct {
	PacketIdentifier uint16
	List             [][]byte
}

// AddTopic adds a given topic that a client wishes to
// unsubscribe from.
func (p *UnsubscribePacket) AddTopic(topic []byte) error {
	// TODO make sure topic is valid
	p.List = append(p.List, topic)
	return nil
}

// Serialize serializes the contents of a unsub packet into
// a []byte buffer.
func (p *UnsubscribePacket) Serialize(b []byte) ([]byte, error) {
	lenPkt := p.Len()
	if b == nil {
		b = make([]byte, lenPkt)
	}
	if len(b) < lenPkt {
		return nil, ErrShortBuffer
	}
	if lenPkt > maxPayloadSize {
		return nil, ErrInvalidPacket
	}
	// write fixed header
	buf := newWritableBuf(b)
	buf.WriteByte(Unsubscribe<<4 | 0x02)
	writePayloadSize(buf, uint32(p.payloadLen()))
	// write packet identifier
	buf.WriteUInt16(p.PacketIdentifier)
	// write topics
	for _, topic := range p.List {
		buf.writeMQTTStr(topic)
	}

	return b[:buf.bytesWritten()], nil
}

func (p *UnsubscribePacket) payloadLen() int {
	payloadLen := 2
	for _, topic := range p.List {
		payloadLen = payloadLen + 2 + len(topic)
	}
	return payloadLen
}

// Len returns number of bytes publish packet will
// take when serialized
func (p *UnsubscribePacket) Len() int {
	payloadLen := p.payloadLen()
	return 1 + // control pkt type + flags
		lenPayloadSizeField(payloadLen) + // remaining length field
		payloadLen
}

// DeserializeUnsubscribePktPayload parses the contents of a bytes slice and returns
// a Unsubscribe as required.
func DeserializeUnsubscribePktPayload(f FixedHeader, p []byte) (*UnsubscribePacket, error) {
	if !f.IsValidFlagsSet() {
		return nil, ErrInvalidPacket
	}

	pkt := &UnsubscribePacket{}
	pr := &pktReader{from: p}
	pkt.PacketIdentifier = pr.readUInt16()
	for !pr.isReadComplete() {
		topic := pr.readStr()
		if err := pkt.AddTopic(topic); err != nil {
			return nil, err
		}
	}
	if pr.err != nil {
		return nil, pr.err
	}
	return pkt, nil
}

// UnsubackPacket is an in-mem representation
// of a unsuback packet
type UnsubackPacket struct {
	PacketIdentifier uint16
}

// Serialize serializes the contents of a unsuback packet into
// a []byte buffer.
func (p *UnsubackPacket) Serialize(b []byte) ([]byte, error) {
	lenPkt := p.Len()
	if b == nil {
		b = make([]byte, lenPkt)
	}
	if len(b) < lenPkt {
		return nil, ErrShortBuffer
	}
	if lenPkt > maxPayloadSize {
		return nil, ErrInvalidPacket
	}
	b[0] = Unsuback<<4 | 0x00
	b[1] = 2
	b[2] = byte(p.PacketIdentifier >> 8)
	b[3] = byte(p.PacketIdentifier)
	return b[:4], nil
}

// Len returns number of bytes packet will
// take when serialized
func (p *UnsubackPacket) Len() int {
	// fixed header(2 bytes) + payload(2)
	return 4
}

// DeserializeUnsubackPktPayload parses the contents of a bytes slice and returns
// a Unsuback packet as required.
func DeserializeUnsubackPktPayload(f FixedHeader, p []byte) (*UnsubackPacket, error) {
	if !f.IsValidFlagsSet() {
		return nil, ErrInvalidPacket
	}
	// payload must be of length 2
	if len(p) != 2 {
		return nil, ErrInvalidPacket
	}
	return &UnsubackPacket{
		PacketIdentifier: uint16(p[0])<<8 + uint16(p[1]),
	}, nil
}
