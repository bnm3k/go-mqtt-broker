package protocol

type connectPacket struct {
	protocolLevel    byte
	usernamePresent  bool
	passwordPresent  bool
	willRetain       bool
	willQoS          byte
	willPresent      bool
	cleanSession     bool
	keepAlive        uint16
	clientIdentifier []byte
	willTopic        []byte
	willMessage      []byte
	username         []byte
	password         []byte
}

type connackPacket struct {
	sessionPresent    bool
	connectReturnCode byte
}

type disconnectPacket struct{}
