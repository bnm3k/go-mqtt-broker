package protocol

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectPacket(t *testing.T) {
	t.Run("everything in connect packet set, happy path", func(t *testing.T) {
		cfg := &ConnectPacketConfig{
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

		pktA, err := NewConnectPacket(cfg)
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// check fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)
		require.True(t, f.IsValidFlagsSet())

		// check payload
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		pktB, err := DeserializeConnectPktPayload(f, payload)
		require.NoError(t, err)
		require.NotNil(t, pktB, "Deserialized connect pkt should not be nil")
		require.Equal(t, pktA, pktB, "Deserialized connect pkt does not match original pkt")
	})

	t.Run("cleanSession set to false but Client ID NOT provided", func(t *testing.T) {
		cfg := &ConnectPacketConfig{
			ClientIdentifier:   []byte("foo"),
			ShouldCleanSession: false,
		}

		pktA, err := NewConnectPacket(cfg)
		pktA.ClientIdentifier = nil // remove client identifier
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// check fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// check payload, should be an error
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("flags should be set to reserved values", func(t *testing.T) {
		pktA, err := NewConnectPacket(&ConnectPacketConfig{ShouldCleanSession: true})
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// set flags to invalid value
		serialized[0] = (Connect << 4) | 4

		// check fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// check payload, should be an error
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("protocol name and version should be valid", func(t *testing.T) {
		pktA, err := NewConnectPacket(&ConnectPacketConfig{ShouldCleanSession: true})
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// check fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// set invalid protocol name & check err
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		copy(payload[:7], []byte{0, 4, 'A', 'B', 'C', 'D', 0x04})
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)

		// set invalid protocol version & check err
		copy(payload[:7], protocolVersion)
		copy(payload[:7], []byte{0, 4, 'M', 'Q', 'T', 'T', 0x09})
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("reserved connect flag should be set to 0", func(t *testing.T) {
		pktA, err := NewConnectPacket(&ConnectPacketConfig{ShouldCleanSession: true})
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// set reserved connect flag to 1 and check error
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		payload[7] = payload[7] | 0x01
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("if will flag is set to 0, then willQoS and willRetain must be non zero", func(t *testing.T) {
		pktA, err := NewConnectPacket(&ConnectPacketConfig{ShouldCleanSession: true})
		require.NoError(t, err)

		pktA.WillFlag = false
		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// set will QoS and willRetain to nonzero and check error
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		payload[7] = payload[7] | 0x30
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("willQoS should be 0,1 or 2", func(t *testing.T) {
		pktA, err := NewConnectPacket(&ConnectPacketConfig{ShouldCleanSession: true})
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// set will QoS to 3 and check error
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		payload[7] = payload[7] | 0x18
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("if username flag set, it should be provided", func(t *testing.T) {
		pktA, err := NewConnectPacket(&ConnectPacketConfig{ShouldCleanSession: true})
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// set username flag and check error
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		payload[7] = payload[7] | 0x80
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("if password flag set, it should be provided", func(t *testing.T) {
		pktA, err := NewConnectPacket(&ConnectPacketConfig{ShouldCleanSession: true, Username: []byte("userA")})
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// set password flag and check error
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		payload[7] = payload[7] | 0x40
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("if password flag set but no username provided, should be protocol violation", func(t *testing.T) {
		pktA, err := NewConnectPacket(&ConnectPacketConfig{ShouldCleanSession: true})
		require.NoError(t, err)

		pktA.passwordPresent = true
		pktA.Password = []byte("password")
		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// check err
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		payload[7] = payload[7] | 0x40
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("if will flag set, will topic and will message should be provided", func(t *testing.T) {
		pktA, err := NewConnectPacket(&ConnectPacketConfig{
			ClientIdentifier:   []byte("abcde"),
			Username:           []byte("foo"),
			Password:           []byte("bar"),
			KeepAliveSeconds:   90,
			ShouldCleanSession: true,
		})
		require.NoError(t, err)

		serialized, err := pktA.Serialize(make([]byte, pktA.Len()))
		require.NoError(t, err)
		require.Equal(t, pktA.Len(), len(serialized))

		// fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connect)

		// set will flag and check for err
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		payload[7] = payload[7] | 0x04
		_, err = DeserializeConnectPktPayload(f, payload)
		require.Error(t, err)
	})
}

func TestConnackPacket(t *testing.T) {
	t.Run("everything ok", func(t *testing.T) {
		pkt := &ConnackPacket{
			SessionPresent: false,
			Code:           ConnAccepted,
		}
		serialized, err := pkt.Serialize(nil)
		require.NoError(t, err)
		require.NotNil(t, serialized)

		// check fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connack)
		require.True(t, f.IsValidFlagsSet())

		// check payload
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		pktRcvd, err := DeserializeConnackPktPayload(f, payload)
		require.NoError(t, err)
		require.NotNil(t, pktRcvd, "Deserialized connect pkt should not be nil")
		require.Equal(t, pkt, pktRcvd, "Deserialized connect pkt does not match original pkt")
	})

	t.Run("payload shorter than 2 bytes", func(t *testing.T) {
		pkt := &ConnackPacket{}
		serialized, err := pkt.Serialize(nil)
		require.NoError(t, err)
		require.NotNil(t, serialized)

		// check fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connack)
		require.True(t, f.IsValidFlagsSet())

		// check payload
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		_, err = DeserializeConnackPktPayload(f, payload[:1])
		require.Error(t, err)
	})

	t.Run("ctrl flags should be set to reserved value", func(t *testing.T) {
		pkt := &ConnackPacket{}
		serialized, err := pkt.Serialize(nil)
		require.NoError(t, err)
		require.NotNil(t, serialized)

		// set invalid ctrl flags
		serialized[0] = serialized[0] | 0x0A

		// check fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, f.PktType, Connack)
		require.False(t, f.IsValidFlagsSet())

		// check payload
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		_, err = DeserializeConnackPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("connect return code should not use reserved values", func(t *testing.T) {
		pkt := &ConnackPacket{
			Code: 10,
		}
		serialized, err := pkt.Serialize(nil)
		require.NoError(t, err)
		require.NotNil(t, serialized)

		// check fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, Connack, f.PktType)
		require.True(t, f.IsValidFlagsSet())

		// check payload
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		_, err = DeserializeConnackPktPayload(f, payload)
		require.Error(t, err)
	})

	t.Run("first 7 bits for connect ack flags should be 0", func(t *testing.T) {
		pkt := &ConnackPacket{}
		serialized, err := pkt.Serialize(nil)
		require.NoError(t, err)
		require.NotNil(t, serialized)

		// check fixed header
		f, err := ReadFixedHeader(bytes.NewReader(serialized))
		require.Equal(t, Connack, f.PktType)
		require.True(t, f.IsValidFlagsSet())

		// set connect ack flags to non-zero and check payload
		payload := serialized[len(serialized)-int(f.PayloadSize):]
		payload[0] = payload[0] | 0xFF
		_, err = DeserializeConnackPktPayload(f, payload)
		require.Error(t, err)
	})
}

func TestDisconnectPacket(t *testing.T) {
	pkt := &DisconnectPacket{}
	serialized, err := pkt.Serialize(nil)
	require.Nil(t, err)
	require.NotNil(t, serialized)
	require.Equal(t, pkt.Len(), len(serialized))

	// read fixed header
	f, err := ReadFixedHeader(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, uint32(0), f.PayloadSize)
	require.Equal(t, Disconnect, f.PktType)
	require.True(t, f.IsValidFlagsSet())
}

func TestPingreqtPacket(t *testing.T) {
	pkt := &PingreqPacket{}
	serialized, err := pkt.Serialize(nil)
	require.Nil(t, err)
	require.NotNil(t, serialized)
	require.Equal(t, pkt.Len(), len(serialized))

	// read fixed header
	f, err := ReadFixedHeader(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, uint32(0), f.PayloadSize)
	require.Equal(t, Pingreq, f.PktType)
	require.True(t, f.IsValidFlagsSet())
}

func TestPingresptPacket(t *testing.T) {
	pkt := &PingrespPacket{}
	serialized, err := pkt.Serialize(nil)
	require.Nil(t, err)
	require.NotNil(t, serialized)
	require.Equal(t, pkt.Len(), len(serialized))

	// read fixed header
	f, err := ReadFixedHeader(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, uint32(0), f.PayloadSize)
	require.Equal(t, Pingresp, f.PktType)
	require.True(t, f.IsValidFlagsSet())
}
