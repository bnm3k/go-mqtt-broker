package protocol

// Packet represents all the methods a packet should implement
type Packet interface {
	Serialize(b []byte) ([]byte, error)
	Len() int
}
