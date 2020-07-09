package protocol

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscribePacket(t *testing.T) {
	pkt := &SubscribePacket{PacketIdentifier: 9999}
	pkt.AddTopic([]byte("foo/bar"), 0)
	pkt.AddTopic([]byte("asd/fgh/jkl"), 1)
	pkt.AddTopic([]byte("qwe/rty/yui/p"), 2)
	serialized, err := pkt.Serialize(nil)
	require.NoError(t, err)
	require.NotNil(t, serialized)

	// check fixed header
	f, err := ReadFixedHeader(bytes.NewReader(serialized))
	require.Equal(t, f.PktType, Subscribe)
	require.True(t, f.IsValidFlagsSet())
	require.Equal(t, uint32(pkt.payloadLen()), f.PayloadSize)

	// check payload
	payload := serialized[len(serialized)-int(f.PayloadSize):]
	pktRcvd, err := DeserializeSubscribePktPayload(f, payload)
	require.NoError(t, err)
	require.NotNil(t, pktRcvd)
	require.Equal(t, pkt, pktRcvd)
}

func TestSubackPacket(t *testing.T) {
	pkt := &SubackPacket{PacketIdentifier: 9999}
	pkt.AddFailure()
	pkt.AddQoSGranted(1)
	pkt.AddQoSGranted(2)
	pkt.AddQoSGranted(0)
	serialized, err := pkt.Serialize(nil)
	require.NoError(t, err)
	require.NotNil(t, serialized)

	// check fixed header
	f, err := ReadFixedHeader(bytes.NewReader(serialized))
	require.Equal(t, f.PktType, Suback)
	require.True(t, f.IsValidFlagsSet())
	require.Equal(t, uint32(pkt.payloadLen()), f.PayloadSize)

	// check payload
	payload := serialized[len(serialized)-int(f.PayloadSize):]
	pktRcvd, err := DeserializeSubackPktPayload(f, payload)
	require.NoError(t, err)
	require.NotNil(t, pktRcvd)
	require.Equal(t, pkt, pktRcvd)
}

func TestUnsubscribePacket(t *testing.T) {
	pkt := &UnsubscribePacket{PacketIdentifier: 9999}
	pkt.AddTopic([]byte("foo/bar"))
	pkt.AddTopic([]byte("asd/fgh/jkl"))
	pkt.AddTopic([]byte("qwe/rty/yui/p"))
	serialized, err := pkt.Serialize(nil)
	require.NoError(t, err)
	require.NotNil(t, serialized)

	// check fixed header
	f, err := ReadFixedHeader(bytes.NewReader(serialized))
	require.Equal(t, f.PktType, Unsubscribe)
	require.True(t, f.IsValidFlagsSet())
	require.Equal(t, uint32(pkt.payloadLen()), f.PayloadSize)

	// check payload
	payload := serialized[len(serialized)-int(f.PayloadSize):]
	pktRcvd, err := DeserializeUnsubscribePktPayload(f, payload)
	require.NoError(t, err)
	require.NotNil(t, pktRcvd)
	require.Equal(t, pkt, pktRcvd)
}

func TestUnsubackPacket(t *testing.T) {
	pkt := &UnsubackPacket{PacketIdentifier: 9999}
	serialized, err := pkt.Serialize(nil)
	require.NoError(t, err)
	require.NotNil(t, serialized)

	// check fixed header
	f, err := ReadFixedHeader(bytes.NewReader(serialized))
	require.Equal(t, f.PktType, Unsuback)
	require.True(t, f.IsValidFlagsSet())
	require.Equal(t, uint32(2), f.PayloadSize)

	// check payload
	payload := serialized[len(serialized)-int(f.PayloadSize):]
	pktRcvd, err := DeserializeUnsubackPktPayload(f, payload)
	require.NoError(t, err)
	require.NotNil(t, pktRcvd)
	require.Equal(t, pkt, pktRcvd)
}
