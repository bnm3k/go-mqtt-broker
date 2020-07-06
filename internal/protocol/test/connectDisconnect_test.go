package protocol_test

import (
	"bytes"
	"testing"

	"github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
	"github.com/stretchr/testify/require"
)

func TestConnectPacket(t *testing.T) {
	t.Run("everything in connect packet set", func(t *testing.T) {
		cfg := &protocol.ConnectPacketConfig{
			ClientIdentifier:   []byte("abcde"),
			Username:           []byte("foo"),
			Password:           []byte("bar"),
			KeepAliveSeconds:   90,
			ShouldCleanSession: true,
			WillTopic:          []byte("buz"),
			WillMessage:        []byte("quz"),
			WillQoS:            2,
			WillRetain:         false,
		}

		pktA, err := protocol.NewConnectPacket(cfg)
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// check fixed header
		pktType, flags, payloadSize, err := protocol.ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, pktType, protocol.Connect)
		require.True(t, protocol.IsValidFlagsSet(pktType, flags))

		// check payload
		payload := serialized[len(serialized)-int(payloadSize):]
		pktB, err := protocol.DeserializeConnectPktPayload(payload)
		require.NoError(t, err)
		require.NotNil(t, pktB, "Deserialized connect pkt should not be nil")
		require.Equal(t, pktA, pktB, "Deserialized connect pkt does not match original pkt")
	})

	t.Run("cleanSession set to false but Client ID NOT provided", func(t *testing.T) {
		cfg := &protocol.ConnectPacketConfig{
			ShouldCleanSession: false,
		}

		pktA, err := protocol.NewConnectPacket(cfg)
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// check fixed header
		pktType, flags, payloadSize, err := protocol.ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, pktType, protocol.Connect)
		require.True(t, protocol.IsValidFlagsSet(pktType, flags))

		// check payload, should be an error
		payload := serialized[len(serialized)-int(payloadSize):]
		_, err = protocol.DeserializeConnectPktPayload(payload)
		require.Error(t, err)
	})
}
