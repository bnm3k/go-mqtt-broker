package broker

import (
	"context"
	"sync"
	"testing"
	"time"

	p "github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
	"github.com/stretchr/testify/assert"
)

func TestFeed(t *testing.T) {
	var feed = NewFeed()
	var done, subscribed sync.WaitGroup
	pkt := &p.PublishPacket{
		QoS:              1,
		PacketIdentifier: 10,
		Dup:              true,
		Retain:           true,
		TopicName:        []byte("foo/bar/baz"),
		Payload:          []byte("abcde"),
	}

	subscriber := func(i int) {
		defer done.Done()
		ch := make(chan *p.PublishPacket)
		sub := feed.Subscribe(ch)
		subscribed.Done()

		timeout := time.NewTimer(2 * time.Second)
		defer timeout.Stop()
		select {
		case v := <-ch:
			assert.Equal(t, pkt, v)
		case <-timeout.C:
			t.Errorf("%d: receive timeout", i)
		}

		sub.Unsubscribe()
	}

	const n = 1000
	done.Add(n)
	subscribed.Add(n)
	for i := 0; i < n; i++ {
		go subscriber(i)
	}
	subscribed.Wait()
	ctx := context.Background()
	// first send
	nsent := feed.Publish(ctx, pkt)
	assert.Equal(t, n, nsent)

	// after first send, each subscriber unsubs
	nsent = feed.Publish(ctx, pkt)
	assert.Equal(t, 0, nsent)
	done.Wait()
}

func TestUnsubscribeFeed(t *testing.T) {
	ctx := context.Background()
	t.Run("Unsubscribe from pending", func(t *testing.T) {
		var (
			feed = NewFeed()
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
			feed = NewFeed()
			ch1  = make(chan *p.PublishPacket)
			ch2  = make(chan *p.PublishPacket)
			sub1 = feed.Subscribe(ch1)
			sub2 = feed.Subscribe(ch2)
			wg   sync.WaitGroup
		)
		defer sub2.Unsubscribe()

		wg.Add(1)
		go func() {
			feed.Publish(ctx, nil)
			wg.Done()
		}()

		// receive on Ch1 then unsubscribe
		<-ch1
		assert.Equal(t, 0, len(feed.pendingSubs))
		assert.Equal(t, 4, len(feed.cases))
		sub1.Unsubscribe()
		assert.Equal(t, 3, len(feed.cases))

		// receive on Ch2
		<-ch2
		wg.Wait()

		// publish again
		wg.Add(1)
		go func() {
			feed.Publish(ctx, nil)
			wg.Done()
		}()
		<-ch2
		assert.Equal(t, 3, len(feed.cases))
		sub2.Unsubscribe()
		assert.Equal(t, 2, len(feed.cases))
		wg.Wait()
	})
}
