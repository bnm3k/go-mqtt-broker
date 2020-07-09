package protocol

// PublishPacket is an in-mem representation
// of a Publish packet
type PublishPacket struct {
	Dup              bool
	QoS              byte
	Retain           bool
	TopicName        []byte
	PacketIdentifier uint16
	Payload          []byte
}

// Serialize serializes the contents of a publish packet into
// a []byte buffer. Buffer should be of appropriate length
// otherwise a ErrShortBuffer error is returned. If nil buffer
// is provided, Serialize instantiates a buffer of required length
// and returns it
func (p *PublishPacket) Serialize(b []byte) ([]byte, error) {
	lenPublishPacket := p.Len()
	if b == nil {
		b = make([]byte, lenPublishPacket)
	}
	if len(b) < lenPublishPacket {
		return nil, ErrShortBuffer
	}
	if lenPublishPacket > maxPayloadSize {
		return nil, ErrInvalidPacket
	}
	// setup ctrl flags
	var ctrlFlag byte = p.QoS << 1
	if p.Dup {
		ctrlFlag = ctrlFlag | 0x08
	}
	if p.Retain {
		ctrlFlag = ctrlFlag | 0x01
	}
	// write fixed header
	buf := newWritableBuf(b)
	buf.WriteByte(Publish<<4 | ctrlFlag)
	writePayloadSize(buf, uint32(p.payloadLen()))

	// write topic name
	buf.writeMQTTStr(p.TopicName)

	// write packet identifier if QoS > 0
	if p.QoS > 0 {
		buf.WriteByte(byte(p.PacketIdentifier >> 8))
		buf.WriteByte(byte(p.PacketIdentifier))
	}
	// write Payload
	buf.Write(p.Payload)

	return b[:buf.bytesWritten()], nil
}

// DeserializePublishPktPayload parses the contents of a bytes slice and returns
// a PublishPacket as required.
func DeserializePublishPktPayload(f FixedHeader, p []byte) (*PublishPacket, error) {
	// parse ctrl flags
	isDuplicate := (f.CtrlFlags & 0x08) != 0
	QoS := (f.CtrlFlags & 0x06) >> 1
	shouldRetain := (f.CtrlFlags & 0x01) != 0

	pr := &pktReader{from: p}

	// get topic name, ensure it is valid?
	topicName := pr.readStr()
	variableHeaderLen := len(topicName) + 2

	// get packet identifier if QoS > 0
	var packetIdentifier uint16 = 0
	if QoS > 0 {
		packetIdentifier = pr.readNum()
		variableHeaderLen += 2
	}

	// get payload
	payloadLen := len(p) - variableHeaderLen
	payload := pr.readBuf(payloadLen)

	if pr.err != nil {
		return nil, pr.err
	}

	pkt := &PublishPacket{
		Dup:              isDuplicate,
		QoS:              QoS,
		Retain:           shouldRetain,
		TopicName:        topicName,
		PacketIdentifier: packetIdentifier,
		Payload:          payload,
	}

	return pkt, nil
}

func (p *PublishPacket) payloadLen() int {
	payloadLen := len(p.TopicName) + 2 + len(p.Payload)
	if p.QoS > 0 {
		// for packet identifier
		payloadLen += 2
	}
	return payloadLen
}

// Len returns number of bytes publish packet will
// take when serialized
func (p *PublishPacket) Len() int {
	payloadLen := p.payloadLen()
	return 1 + // control pkt type + flags
		lenPayloadSizeField(payloadLen) + // remaining length field
		payloadLen
}

type pubackPacket struct {
	packetIdentifier uint16
}

type pubrecPacket struct {
	packetIdentifier uint16
}

type pubrelPacket struct {
	packetIdentifier uint16
}

type pubcompPacket struct {
	packetIdentifier uint16
}
