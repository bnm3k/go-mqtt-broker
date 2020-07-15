package broker

import (
	"bufio"
	"net"
	"sync"
	"testing"

	"github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
	"github.com/stretchr/testify/require"
)

func TestBroker(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	serverSide, clientSide := net.Pipe()
	go func() {
		broker := NewBroker()
		broker.OnConn(serverSide)
		wg.Done()
	}()

	// Client side
	cfg := &protocol.ConnectPacketConfig{
		ClientIdentifier:   []byte("abcde"),
		KeepAliveSeconds:   1,
		ShouldCleanSession: true,
	}

	// send connect packet
	connectPkt, err := protocol.NewConnectPacket(cfg)
	require.NoError(t, err)
	buf, err := connectPkt.Serialize(nil)
	require.NoError(t, err)
	n, err := clientSide.Write(buf)
	require.NoError(t, err)
	require.Equal(t, len(buf), n)

	// receive connack packet
	clientSideRead := mqttPacketReader{bufio.NewReader(clientSide)}
	f, p, err := clientSideRead.readPkt()
	require.NoError(t, err)
	require.Equal(t, protocol.Connack, f.PktType)
	require.True(t, f.IsValidFlagsSet())
	connackPkt, err := protocol.DeserializeConnackPktPayload(f, p)
	require.NoError(t, err)
	require.NotNil(t, connackPkt)
	require.Equal(t, protocol.ConnAccepted, connackPkt.Code)

	// send ping req packet
	pingReqPkt := protocol.PingreqPacket{}
	buf, err = (&pingReqPkt).Serialize(nil)
	require.NoError(t, err)
	n, err = clientSide.Write(buf)
	require.NoError(t, err)
	require.Equal(t, len(buf), n)

	// receive ping resp packet
	f, p, err = clientSideRead.readPkt()
	require.NoError(t, err)
	require.Equal(t, protocol.Pingresp, f.PktType)
	require.True(t, f.IsValidFlagsSet())
	require.True(t, f.PayloadSize == 0)
	require.True(t, p == nil || len(p) == 0)

	// send disconnect packet
	disconnectPkt := protocol.DisconnectPacket{}
	buf, err = (&disconnectPkt).Serialize(nil)
	require.NoError(t, err)
	n, err = clientSide.Write(buf)
	require.NoError(t, err)
	require.Equal(t, len(buf), n)

	// next write should fail
	n, err = clientSide.Write(buf)
	require.Error(t, err)

	// end connection
	clientSide.Close()
	wg.Wait()
}
