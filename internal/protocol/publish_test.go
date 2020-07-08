package protocol

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublishPacket(t *testing.T) {
	t.Run("everything set correct", func(t *testing.T) {
		pkt := &PublishPacket{
			QoS:              1,
			PacketIdentifier: 10,
			Dup:              true,
			Retain:           true,
			TopicName:        []byte("foo/bar/baz"),
			Payload:          []byte("abcde"),
		}

		serialized, err := pkt.Serialize(nil)
		require.NoError(t, err)
		require.NotNil(t, serialized)
		require.Equal(t, pkt.Len(), len(serialized))

		// check fixed header
		pktType, ctrlFlags, payloadSize, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, pktType, Publish)
		require.True(t, isValidFlagsSet(pktType, ctrlFlags))

		// check payload
		payload := serialized[len(serialized)-int(payloadSize):]
		pktDs, err := DeserializePublishPktPayload(ctrlFlags, payload)
		require.NoError(t, err)
		require.NotNil(t, pktDs, "Deserialized publish pkt should not be nil")
		require.Equal(t, pkt, pktDs, "Deserialized publish pkt does not match original pkt")
	})

	t.Run("QoS is zero but packet identifier provided", func(t *testing.T) {
		t.Skip()
		pkt := &PublishPacket{
			QoS:              1,
			PacketIdentifier: 999,
			TopicName:        []byte("a/b"),
			Payload:          []byte("abcde"),
		}

		serialized, err := pkt.Serialize(nil)
		require.NoError(t, err)
		require.NotNil(t, serialized)
		require.Equal(t, pkt.Len(), len(serialized))

		// set QoS to 0
		serialized[0] = serialized[0] & 0xF9

		// check fixed header
		pktType, ctrlFlags, payloadSize, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, pktType, Publish)
		require.True(t, isValidFlagsSet(pktType, ctrlFlags))

		// check payload
		payload := serialized[len(serialized)-int(payloadSize):]
		_, err = DeserializePublishPktPayload(ctrlFlags, payload)
		require.Error(t, err)
	})

}
