package broker

import (
	"bufio"
	"io"
	"net"

	p "github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
)

type clientSession struct {
	conn net.Conn
	id   string
}

func (c *clientSession) readPackets() {
	r := mqttPacketReader{bufio.NewReader(c.conn)}
	for {
		f, payload, err := r.readPkt()
		if err != nil {
			return
		}
		c.handlePacket(f, payload)
	}
}

func (c *clientSession) sendPacket(pkt p.Packet) (err error) {
	var p []byte
	p, err = pkt.Serialize(nil)
	if err == nil {
		_, err = c.conn.Write(p)
	}
	return
}

func (c *clientSession) handlePacket(f p.FixedHeader, payload []byte) {

}

type mqttPacketReader struct {
	r *bufio.Reader
}

func (r mqttPacketReader) readPkt() (f p.FixedHeader, payload []byte, err error) {
	// read fixed header
	f, err = p.ReadFixedHeader(r.r)
	if err != nil {
		return
	}
	// read rest of payload
	if f.PayloadSize > 0 {
		payload = make([]byte, f.PayloadSize)
		_, err = io.ReadFull(r.r, payload)
		if err != nil {
			return
		}
	}
	return
}
