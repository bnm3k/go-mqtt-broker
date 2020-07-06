package broker

import (
	"bufio"
	"io"
	"net"
	"sync"

	"github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
)

// Broker encapsulates all the functionality of a MQTT broker plus
// rules. It also holds shared resources such as topics or client IDs
// that've been issued
type Broker struct {
	clientsWg sync.WaitGroup
	onceClose sync.Once
	quitCh    chan struct{}
}

// OnConn is an implementation of the server's ConnHandler.OnConn
// Should be called whenever there's a new connection. If the connection
// is valid, it is elevated into a ClientSession. If it's invalid or an error
// occurs such as a protocol violation, the connection is closed
func (b *Broker) OnConn(conn net.Conn) {
	select {
	case <-b.quitCh:
		return
	default:
		b.clientsWg.Add(1) // add new client conn
		go func() {
			b.handleConn(conn)
			b.clientsWg.Done() // indicate client done
		}()
	}

}

func (b *Broker) handleConn(conn net.Conn) {
	// conn.SetDeadline(time.Now().Add(1 * time.Second))
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		var buf []byte
		// read fixed header
		_, payloadSize, err := protocol.ReadFixedHeader(r)
		if err != nil {
			return
		}
		// read rest of payload
		if payloadSize > 0 {
			buf = make([]byte, payloadSize)
			_, err = io.ReadFull(r, buf)
			if err != nil {
				return
			}
		}
	}
}

// Close is an implementation of the server's ConnHandler.Close. It's
// expected that the server instance will invoke Close when it too is closed
// however, Close is safe to call multiple times. Once closed, the broker will
// not accept any more connections. It's expected that each underlying client session
// will also gracefully shut down
func (b *Broker) Close() {
	b.onceClose.Do(func() {
		close(b.quitCh)
		b.clientsWg.Wait()
	})
}
