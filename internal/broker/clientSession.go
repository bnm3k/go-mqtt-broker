package broker

import (
	"bufio"
	"io"
	"net"
	"sync"

	p "github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
)

type clientSession struct {
	conn       net.Conn
	id         string
	broker     *Broker
	onceClose  sync.Once
	closeSigCh chan struct{}
}

func newClientSession(id string, conn net.Conn) *clientSession {
	return &clientSession{
		id:         id,
		conn:       conn,
		closeSigCh: make(chan struct{}),
	}
}

func (c *clientSession) start() {
	// handler for incoming pkts
	handlePacket := func(f p.FixedHeader, payload []byte) {
		switch f.PktType {
		case p.Pingreq:
			c.sendPacket(&p.PingrespPacket{})
		case p.Publish:
		case p.Subscribe:
		case p.Pingresp:
		case p.Disconnect:

		}
	}
	// read incoming pkts
	go func() {
		r := mqttPacketReader{bufio.NewReader(c.conn)}
		for {
			f, payload, err := r.readPkt()
			if err != nil {
				c.close()
				return
			}
			handlePacket(f, payload)
		}
	}()

	// monitor
	for {

		select {
		case <-c.closeSigCh:
			return
		}
	}

}

func (c *clientSession) close() {
	c.onceClose.Do(func() {
		close(c.closeSigCh)
	})
}

func (c *clientSession) sendPacket(pkt p.Packet) (err error) {
	var p []byte
	p, err = pkt.Serialize(nil)
	if err == nil {
		_, err = c.conn.Write(p)
	}
	return
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
