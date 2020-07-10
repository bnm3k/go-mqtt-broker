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

	// Do some stuff
	cfg := &protocol.ConnectPacketConfig{
		ClientIdentifier:   []byte("abcde"),
		KeepAliveSeconds:   90,
		ShouldCleanSession: true,
	}

	connectPkt, err := protocol.NewConnectPacket(cfg)
	require.NoError(t, err)

	buf, err := connectPkt.Serialize(nil)
	require.NoError(t, err)
	clientSide.Write(buf)

	clientSideRead := mqttPacketReader{bufio.NewReader(clientSide)}
	f, p, err := clientSideRead.readPkt()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, protocol.Connack, f.PktType)
	require.True(t, f.IsValidFlagsSet())
	connackPkt, err := protocol.DeserializeConnackPktPayload(f, p)
	require.NoError(t, err)
	require.NotNil(t, connackPkt)
	require.Equal(t, protocol.ConnAccepted, connackPkt.Code)

	clientSide.Close()
	wg.Wait()
}
