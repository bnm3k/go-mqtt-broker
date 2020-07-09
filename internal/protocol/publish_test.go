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
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Publish)
		require.True(t, f.IsValidFlagsSet())

		// check payload
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		pktDs, err := DeserializePublishPktPayload(f, payload)
		require.NoError(t, err)
		require.NotNil(t, pktDs, "Deserialized publish pkt should not be nil")
		require.Equal(t, pkt, pktDs, "Deserialized publish pkt does not match original pkt")
	})

	t.Run("QoS is zero but packet identifier provided", func(t *testing.T) {
		// find way to test for this?
	})

	t.Run("if QoS is 0, then dup flag must be 0", func(t *testing.T) {
		// find way to test for this?
	})

	t.Run("publish flag must not have both QoS bits set to 1, ie QoS of value 3", func(t *testing.T) {
		// this should be handled by use of fixedHeader.IsValidFlagsSet()
	})

	t.Run("topic name should be valid, plus not contain any wildcard characters", func(t *testing.T) {
		// this should be handled by use of fixedHeader.IsValidFlagsSet()
	})

}
