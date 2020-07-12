package broker

import (
	"context"
	"reflect"
	"sync"

	p "github.com/nagamocha3000/go-mqtt-broker/internal/protocol"
)

// PublishPacket ...
type PublishPacket struct {
	msg string
}

// Feed ...
type Feed struct {
	// holds currently subscribed channels, whenever modifying
	// or accessing cases, one should acquire the sendLock
	sendLock chan struct{}
	cases    []reflect.SelectCase

	// for remove during a send (interrupts)
	removeSubCh chan *sub

	// holds newly subsribed channels until they are added to cases
	pendingMu   sync.Mutex
	pendingSubs []reflect.SelectCase
}

const firstSubSendCase = 2

var emptySelectCase reflect.SelectCase

//NewFeed ...
func NewFeed() *Feed {
	f := new(Feed)
	f.removeSubCh = make(chan *sub)
	f.sendLock = make(chan struct{}, 1)
	f.sendLock <- struct{}{}
	f.cases = []reflect.SelectCase{
		emptySelectCase,
		{Chan: reflect.ValueOf(f.removeSubCh), Dir: reflect.SelectRecv},
	}
	return f
}

// Subscribe ...
func (f *Feed) Subscribe(ch chan *p.PublishPacket) Subscription {
	sub := &sub{
		feed:    f,
		channel: ch,
	}

	// add to pending, will be added on next send
	f.pendingMu.Lock()
	defer f.pendingMu.Unlock()
	f.pendingSubs = append(f.pendingSubs, reflect.SelectCase{
		Dir:  reflect.SelectSend,
		Chan: reflect.ValueOf(ch),
	})

	return sub
}

// Remove ...
func (f *Feed) remove(sub *sub) {
	// if in pending, delete first
	f.pendingMu.Lock()
	for i := 0; i < len(f.pendingSubs); i++ {
		if f.pendingSubs[i].Chan.Interface() == sub.channel {
			f.pendingSubs = caseDelete(f.pendingSubs, i)
			f.pendingMu.Unlock()
			return
		}
	}
	f.pendingMu.Unlock()

	// otherwise ...
	select {
	case f.removeSubCh <- sub:
		// send will remove the channel
	case <-f.sendLock:
		i := caseFindIndex(f.cases, sub.channel)
		f.cases = caseDelete(f.cases, i)
		f.sendLock <- struct{}{}
	}

}

// Publish ...
func (f *Feed) Publish(ctx context.Context, pkt *p.PublishPacket) (nSent int) {
	<-f.sendLock

	// add new cases from pending subs
	f.pendingMu.Lock()
	f.cases = append(f.cases, f.pendingSubs...)
	f.pendingSubs = nil
	f.pendingMu.Unlock()

	// set up rval & the send on all channels
	rval := reflect.ValueOf(pkt)
	for i := firstSubSendCase; i < len(f.cases); i++ {
		f.cases[i].Send = rval
	}

	// send until all channels have been chosen
	f.cases[0] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	}
	currCases := f.cases

	for {

		// first send to all those that can receive without blocking
		for i := firstSubSendCase; i < len(currCases); i++ {
			if currCases[i].Chan.TrySend(rval) {
				nSent++
				currCases = caseDelete(currCases, i)
				i--
			}
		}

		if len(currCases) == firstSubSendCase {
			break
		}

		// select any of the recepients randomly that's ready to receive
		chosen, recv, _ := reflect.Select(currCases)
		if chosen == 0 { // <-ctx.Done()
			break
		} else if chosen == 1 { // <-f.removeSub
			sub := recv.Interface().(*sub)
			index := caseFindIndex(f.cases, sub.channel)
			f.cases = caseDelete(f.cases, index)
			if index >= 0 && index < len(currCases) {
				currCases = caseDelete(currCases, index)
			}
		} else {
			currCases = caseDelete(currCases, chosen)
			nSent++
		}

	}

	// forget about send val, hand off sendLock
	for i := 1; i < len(f.cases); i++ {
		f.cases[i].Send = reflect.Value{}
	}
	f.cases[0] = emptySelectCase
	f.sendLock <- struct{}{}
	return nSent
}

func caseFindIndex(cs []reflect.SelectCase, ch chan *p.PublishPacket) int {
	for i := firstSubSendCase; i < len(cs); i++ {
		if cs[i].Chan.Interface() == ch {
			return i
		}
	}
	return -1
}

func caseDelete(cs []reflect.SelectCase, index int) []reflect.SelectCase {
	last := len(cs) - 1
	cs[index], cs[last] = cs[last], cs[index]
	return cs[:last]
}

// Subscription ...
type Subscription interface {
	Unsubscribe()
}

type sub struct {
	feed            *Feed
	channel         chan *p.PublishPacket
	onceUnsubscribe sync.Once
}

func (s *sub) Unsubscribe() {
	s.onceUnsubscribe.Do(func() {
		s.feed.remove(s)
	})
}
