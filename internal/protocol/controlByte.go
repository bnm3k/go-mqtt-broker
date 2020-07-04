package protocol

type controlPacketType uint8

const (
	connect uint8 = iota + 1
	connack
	publish
	puback
	pubrec
	pubrel
	pubcomp
	subscribe
	suback
	unsubscribe
	unsuback
	pingreq
	pingresp
	disconnect
)

func (c controlPacketType) String() string {
	return [...]string{
		"Reserved", "CONNECT", "CONNACK",
		"PUBLISH", "PUBACK", "PUBREC",
		"PUBREL", "PUBCOMP", "SUBSCRIBE",
		"SUBACK", "UNSUBSCRIBE", "UNSUBACK",
		"PINGREQ", "PINGRESP", "DISCONNECT", "Reserved"}[c]
}

func getReservedFlags(c uint8) uint8 {
	if c == pubrel || c == subscribe || c == unsubscribe {
		return 0x02
	} // for rest kind of packets
	return 0x00
}

func isValidControlPacketType(i uint8) bool {
	return i >= 0 && i <= 15
}

func isReservedControlPacketType(i uint8) bool {
	return i == 0 || i == 15
}

func isValidFlagsSet(ctrlPktType uint8, flag uint8) bool {
	// check section 2.2.2 on the default flags to be set
	switch ctrlPktType {
	case publish:
		// check section 3.3.1.2 on QoS
		// from spec: A PUBLISH Packet MUST NOT have both QoS
		// bits set to 1. If a Server or Client receives a PUBLISH
		// Packet which has both QoS bits set to 1 it MUST close
		// the Network Connection
		return flag|0x06 != 0x06
	case pubrel, subscribe, unsubscribe:
		return flag == 0x02
	default:
		return flag == 0x00
	}
}

// for use with non-publish type packets, if used with
// publish type packet, all flags set to 0, which
// means no duplication, At most once delivery and no retain
func serializeControlPacket(ctrlPktType uint8) uint8 {
	var flags uint8
	// set flags to required reserved type
	// 0x00, 0x0F are reserved
	check(ctrlPktType <= 0x0F, "invalid ctrlPktType")
	check(ctrlPktType != 0x00 && ctrlPktType != 0x0F, "invalid ctrlPktType, reserved")
	switch ctrlPktType {
	case pubrel, subscribe, unsubscribe:
		flags = 0x02
	default:
		flags = 0x00
	}
	return (ctrlPktType << 4) | flags
}

// flag ORred
const publishFlagDup = 0x08
const publishFlagQOSAtMostOnce = 0x00
const publishFlagQOSAtLeastOnce = 0x02
const publishFlagQOSExactlyOnce = 0x04
const publishFlagQOSReserved = 0x06
const publishFlagRetain = 0x01

func serializeControlPacketPublish(qos uint8, setDup, setRetain bool) uint8 {
	var ctrlPktType uint8 = publish
	// assert qos is valid
	check(qos == 0x00 || qos == 0x02 || qos == 0x04, "is invalid qos")
	var flags uint8 = qos
	if setDup {
		flags = flags | publishFlagDup
	}
	if setRetain {
		flags = flags | publishFlagRetain
	}
	return (ctrlPktType << 4) | flags
}

// returns the control packet type and flags set, more of a helper
func deserializeControlPacket(ctrlPkt uint8) (uint8, uint8) {
	return ctrlPkt >> 4, ctrlPkt & 0x0F
}

/*
FROM NETWORK
-> read fixed header
-> read rest of payload based on payload size
-> get ctrl Byte from fixed header + payload and send to handler

TOP-LEVEL HANDLER (ctrl Byte, payload)
-> get type and flags from fixed header
-> if type is reserved error out
-> otherwise dispatch to the required specific handler

SPECIFIC HANDLER (flags Byte, payload)
-> check flags, should match expected, else error out
->
*/
