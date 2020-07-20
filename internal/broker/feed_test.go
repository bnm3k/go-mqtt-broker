package broker

import (
	"context"
	"sync"
	"testing"
	"time"

	p "github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeed(t *testing.T) {
	var feed = NewFeed("-")
	var done, subscribed, unsubscribed sync.WaitGroup
	pkt := &p.PublishPacket{
		QoS:              1,
		PacketIdentifier: 10,
		Dup:              true,
		Retain:           true,
		TopicName:        []byte("foo/bar/baz"),
		Payload:          []byte("abcde"),
	}

	subscriber := func(i int) {
		ch := make(chan *p.PublishPacket)
		sub := feed.Subscribe(ch)
		subscribed.Done()

		timeout := time.NewTimer(2 * time.Second)
		defer timeout.Stop()
		select {
		case v := <-ch:
			require.Equal(t, pkt, v)
		case <-timeout.C:
			t.Errorf("%d: receive timeout", i)
		}

		sub.Unsubscribe()
		unsubscribed.Done()
		done.Done()
	}

	const n = 1000
	subscribed.Add(n)
	unsubscribed.Add(n)
	done.Add(n)
	require.Equal(t, 0, len(feed.pendingSubs))
	require.Equal(t, 2, len(feed.cases))
	for i := 0; i < n; i++ {
		go subscriber(i)
	}

	subscribed.Wait()
	require.Equal(t, n, len(feed.pendingSubs))
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	// first send
	nsent := feed.Publish(ctx, pkt)
	require.Equal(t, n, nsent)

	unsubscribed.Wait()
	require.Equal(t, 2, len(feed.cases))

	// after first send, each subscriber unsubs
	nsent = feed.Publish(ctx, pkt)
	require.Equal(t, 0, nsent)
	done.Wait()
}

func TestUnsubscribeFeed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pkt := &p.PublishPacket{
		QoS:              1,
		PacketIdentifier: 10,
		Dup:              true,
		Retain:           true,
		TopicName:        []byte("foo/bar/baz"),
		Payload:          []byte("abcde"),
	}

	t.Run("Unsubscribe from pending", func(t *testing.T) {
		var (
			feed = NewFeed("-")
			ch1  = make(chan *p.PublishPacket)
			ch2  = make(chan *p.PublishPacket)
			sub1 = feed.Subscribe(ch1)
			sub2 = feed.Subscribe(ch1)
			sub3 = feed.Subscribe(ch2)
		)

		assert.Equal(t, 3, len(feed.pendingSubs))
		assert.Equal(t, 2, len(feed.cases))

		sub1.Unsubscribe()
		sub2.Unsubscribe()
		sub3.Unsubscribe()

		assert.Equal(t, 0, len(feed.pendingSubs))
		assert.Equal(t, 2, len(feed.cases))
	})

	t.Run("Unsubscribe during sending", func(t *testing.T) {
		var (
			feed = NewFeed("-")
			ch1  = make(chan *p.PublishPacket)
			ch2  = make(chan *p.PublishPacket)
			sub1 = feed.Subscribe(ch1)
			sub2 = feed.Subscribe(ch2)
			wg   sync.WaitGroup
		)

		wg.Add(1)
		go func() {
			nSent := feed.Publish(ctx, pkt)
			require.Equal(t, 2, nSent)
			wg.Done()
		}()

		// receive on Ch1 then unsubscribe
		<-ch1
		require.Equal(t, 0, len(feed.pendingSubs))
		require.Equal(t, 4, len(feed.cases))
		sub1.Unsubscribe()

		// receive on Ch2
		<-ch2
		wg.Wait()
		require.Equal(t, 3, len(feed.cases))

		// publish again
		wg.Add(1)
		go func() {
			nSent := feed.Publish(ctx, pkt)
			require.Equal(t, 1, nSent)
			wg.Done()
		}()
		<-ch2
		require.Equal(t, 3, len(feed.cases))
		sub2.Unsubscribe()
		require.Equal(t, 2, len(feed.cases))
		wg.Wait()
	})
}
