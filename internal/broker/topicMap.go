package broker

import (
	"fmt"
	"sync"

	hm "github.com/cornelk/hashmap"
)

// node holds a level for a wildcard match
type node struct {
	children map[string]*node
	feed     *Feed
}

// TopicMap holds a specialized map of topics to feeds through which subscribers can receive
// publish events. Given a topic T, eg during a publish event, it efficiently finds all
// topics (both exact and thos with wildcards) that match the given topic and returns
// the subsequent Feeds. If the topic T has l levels, then finding matching topics is
// O(l) regardless of the entire length of the topic name or number of matching topics
// This is under the assumption that hashing the string of a single level of constant
// time. TopicMap is concurrency safe, ie can be accessed safely from multiple concurrent
// goroutines.
type TopicMap struct {
	root        *node
	rwLock      *sync.RWMutex
	topicToFeed *hm.HashMap
}

// NewTopicMap returns an instance of a topic map
// for holding topics with wildcards
func NewTopicMap() TopicMap {
	//var rwLock sync.RWMutex
	return TopicMap{
		root: &node{
			children: make(map[string]*node),
		},
		rwLock:      &sync.RWMutex{},
		topicToFeed: &hm.HashMap{},
	}
}

// InitFeedByTopic ensures that the feed for a given topic is
// instantiated. Note that a given topic should have at least 1
// token as per the MQTT requirements. The bool returned indicates
// whether the feed was already present (true) or it's just been
// instantiated (false). Mainly there for debugging/testing.For simplicity,
// one should use the ParseTopic helper to parse a given topic name,
// check for errors then retrieve the appropriate arguments to pass to the function
func (m TopicMap) InitFeedByTopic(topic string, tokens []TopicToken) (*Feed, bool) {

	// first check topic to feed
	if feed, alreadyPresent := m.topicToFeed.Get(topic); alreadyPresent {
		return feed.(*Feed), alreadyPresent
	}

	// if topic feed not set, add
	// lock for writing
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	curr := m.root
	for _, token := range tokens {
		next, present := curr.children[token.Value]
		if !present {
			next = &node{
				children: make(map[string]*node),
			}
			curr.children[token.Value] = next
		}
		curr = next
	}
	// Set topic
	if curr.feed == nil {
		feed := NewFeed(topic)
		curr.feed = feed
		m.topicToFeed.Set(topic, feed)
		return feed, false
	}
	return curr.feed, true

}

// RemoveFeedByTopic removes a given feed. It's
// better to keep feeds in place rather than remove them frequently.
// However, the function is provided for situations whereby
// there are multiple feeds but each is sparsely and infrequently
// used. A Nil feed is returned if the feed was not present to
// begin with. For simplicity, one should use the ParseTopic helper
// to parse a given topic name, check for errors then retrieve the appropriate
// arguments to pass to the function
func (m TopicMap) RemoveFeedByTopic(topic string, tokens []TopicToken) *Feed {
	// lock for writing
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	m.topicToFeed.Del(string(topic))

	curr := m.root
	for _, token := range tokens {
		next, present := curr.children[token.Value]
		if !present {
			return nil
		}
		curr = next
	}
	feed := curr.feed
	curr.feed = nil // GC

	return feed
}

// GetFeedsThatMatchTopic The given topic should be an exact topic match, ie,
// it should not have any wildcards. For use mainly when a publish packet
// arrives and one needs to check whether a subscriber qualifies to receive
// it on the basis of a topic they supplied earlier which contained wildcards.
// topicTokens should consist of valid tokens of a parsed topic and should
// not contain any wildcards since this function is meant to be used when one
// is publishing a publish packet. It is expected that the caller uses the
// helper function ParseTopicName to check for possible errors and retrieve
// valid tokens
func (m TopicMap) GetFeedsThatMatchTopic(topicTokens []TopicToken) []*Feed {
	// lock for reading
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()

	// find matching wildcard topics
	feeds := findFeedsThatMatchTopic(0, topicTokens, m.root, make([]*Feed, 0, 10))

	return feeds
}

func findFeedsThatMatchTopic(level int, tokens []TopicToken, curr *node, feeds []*Feed) []*Feed {
	// first check single level matches
	matches := [2]string{tokens[0].Value, "+"}
	for _, m := range matches {
		if v, ok := curr.children[m]; ok {

			// last token
			if len(tokens) == 1 {
				if v.feed != nil {
					feeds = append(feeds, v.feed)
				}
				// # matches parent level too
				if vp, ok := v.children["#"]; ok && vp.feed != nil {
					feeds = append(feeds, vp.feed)
				}
				continue
			}

			// more remaining tokens
			feeds = findFeedsThatMatchTopic(level+1, tokens[1:], v, feeds)

		}
	}

	// check multi-level matches
	if v, ok := curr.children["#"]; ok && v.feed != nil {
		feeds = append(feeds, v.feed)
	}
	return feeds
}

func printStr(level int, str string) {
	for i := 0; i < level; i++ {
		fmt.Print("   ")
	}
	if level > 0 {
		fmt.Print("\t")
	}
	fmt.Print(str, "\n")
}

func printFeeds(level int, prefix string, fs []*Feed) {
	fmt.Print("\t")
	for i := 0; i < level; i++ {
		fmt.Print("   ")
	}
	fmt.Print(prefix, " ")

	// print feeds
	if len(fs) == 0 {
		fmt.Print("[]")
	} else {
		last := len(fs) - 1
		fmt.Print("[")
		for _, f := range fs[:last] {
			fmt.Print(f.topic, ",  ")
		}
		fmt.Print(fs[last].topic)
		fmt.Print(" ]")
	}
	fmt.Print("\n")
}

// TraverseAll .For debugging mostly
func (m TopicMap) TraverseAll(fn func(int, *node)) {
	// lock for reading
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()

	traverseAll(0, m.root, fn)
}

func traverseAll(level int, n *node, fn func(int, *node)) {
	fn(level, n)

	for _, cn := range n.children {
		traverseAll(level+1, cn, fn)
	}
}
