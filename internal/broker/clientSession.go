package broker

import (
	"bufio"
	"io"
	"net"
	"sync"

	p "github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
)

type clientSession struct {
	closeSigCh    chan struct{}
	conn          net.Conn
	id            string
	subscriptions map[string]Subscription
	topicMap      TopicMap

	willFlag  bool
	onceClose sync.Once
}

func newClientSession(id string, conn net.Conn, tm TopicMap) *clientSession {
	return &clientSession{
		closeSigCh: make(chan struct{}),
		conn:       conn,
		id:         id,
		topicMap:   tm,
	}
}

func (c *clientSession) start() {

	// channel for messages client has subscribed to
	messagesCh := make(chan *p.PublishPacket)

	// handler for incoming pkts
	handlePacket := func(f p.FixedHeader, payload []byte) {
		switch f.PktType {
		case p.Pingreq:
			c.sendPacket(&p.PingrespPacket{})
		case p.Publish:
			_, err := p.DeserializePublishPktPayload(f, payload)
			if err != nil {
				c.close()
				return
			}
		case p.Subscribe:
			_, err := p.DeserializeSubscribePktPayload(f, payload)
			if err != nil {
				c.close()
				return
			}
			// send suback
		case p.Unsubscribe:
			pkt, err := p.DeserializeUnsubscribePktPayload(f, payload)
			if err != nil {
				c.close()
				return
			}
			ackPkt := p.UnsubackPacket{PacketIdentifier: pkt.PacketIdentifier}
			c.sendPacket(&ackPkt)
		case p.Disconnect:
			c.willFlag = false
			c.close()
		default:
			c.close()
		}
	}

	// read incoming pkts
	go func() {
		r := mqttPacketReader{bufio.NewReader(c.conn)}
		for {
			select {
			case <-c.closeSigCh:
				return
			default:
				f, payload, err := r.readPkt()
				if err != nil || !f.IsValidFlagsSet() {
					return
				}
				handlePacket(f, payload)
			}
		}
	}()

	// monitor
	for {
		select {
		case <-messagesCh:
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
