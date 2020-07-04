package protocol

type publishPacket struct {
	dup              bool
	qos              byte
	retain           bool
	topicName        []byte
	packetIdentifier uint16
	payload          []byte
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
