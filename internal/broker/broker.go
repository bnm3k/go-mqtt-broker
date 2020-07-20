package broker

import (
	"bufio"
	"errors"
	"net"
	"sync"
	"time"

	p "github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
	"github.com/rs/xid"
)

// Broker encapsulates all the functionality of a MQTT broker plus
// rules. It also holds shared resources such as topics or client IDs
// that've been issued
type Broker struct {
	clientIDs    map[string]bool
	clientsWg    sync.WaitGroup
	onceClose    sync.Once
	quitCh       chan struct{}
	connDeadline time.Duration
	topicMap     TopicMap
}

// NewBroker returns a fresh instance of a Broker
func NewBroker() *Broker {
	return &Broker{
		clientIDs:    make(map[string]bool),
		quitCh:       make(chan struct{}),
		connDeadline: 1 * time.Second,
		topicMap:     NewTopicMap(),
	}
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
			defer func() {
				conn.Close()
				b.clientsWg.Done() // indicate client done
			}()
			clientSession, err := b.handleNewClientConnection(conn)
			if err != nil {
				// close connection
				return
			}
			clientSession.start()
			// on end, remove client ID
		}()
	}
}

var errConn = errors.New("Client connection error occured")

func (b *Broker) handleNewClientConnection(conn net.Conn) (*clientSession, error) {
	// set deadline
	//conn.SetDeadline(time.Now().Add(b.connDeadline))

	// read first packet, should be connect
	r := mqttPacketReader{bufio.NewReader(conn)}
	f, payload, err := r.readPkt()
	if err != nil {
		return nil, err
	}

	// deserialize
	pkt, err := p.DeserializeConnectPktPayload(f, payload)
	if err != nil {
		return nil, err
	}

	// instantiate client session
	cs := newClientSession(string(pkt.ClientIdentifier), conn, b.topicMap)

	// authenticate
	if ok := b.authenticate(pkt.Username, pkt.Password); !ok {
		cs.sendPacket(&p.ConnackPacket{Code: p.ConnRefusedBadUsernamePass})
		return nil, errConn
	}

	// check given client identifier
	if len(pkt.ClientIdentifier) > 0 {
		if _, ok := b.clientIDs[string(pkt.ClientIdentifier)]; ok {
			cs.sendPacket(&p.ConnackPacket{Code: p.ConnRefusedIdentifierRejected})
			return nil, errConn
		}
	} else { // if no client identifier provided, assign one
		for {
			newID := xid.New().String()
			if _, ok := b.clientIDs[newID]; !ok {
				b.clientIDs[newID] = true
				cs.id = newID
				break
			}
		}
	}

	// unset deadline
	// conn.SetDeadline(time.Time{})

	// Check KeepAlive

	// Check if should clean Session

	// Check will message & topic

	err = cs.sendPacket(&p.ConnackPacket{Code: p.ConnAccepted})
	return cs, err
}

func (b *Broker) authenticate(username, password []byte) bool {
	return true
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
