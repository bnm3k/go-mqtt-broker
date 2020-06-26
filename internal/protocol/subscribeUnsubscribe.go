package protocol

type subscribePacket struct {
	packetIdentifier uint16
	list             []struct {
		topic []byte
		qos   byte
	}
}

type subackPacket struct {
	packetIdentifier uint16
	returnCodes      []byte
}

type unsubscribePacket struct {
	packetIdentifier uint16
	list             [][]byte
}

type unsubackPacket struct {
	packetIdentifier uint16
}
