package broker

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTokenWildcardPermsSingleLevel(tokens []TopicToken) [][]TopicToken {
	// minor todo, make more space efficent
	if len(tokens) == 1 {
		return [][]TopicToken{
			{TopicToken{Value: "+", MatchType: SingleLevelMatch}},
			{tokens[0]},
		}
	}
	var perms [][]TopicToken
	childPerms := generateTokenWildcardPermsSingleLevel(tokens[1:])

	for _, childPerm := range childPerms {
		permWithToken := make([]TopicToken, len(childPerm)+1)
		permWithToken[0] = tokens[0]
		copy(permWithToken[1:], childPerm)
		permWithPlus := make([]TopicToken, len(childPerm)+1)
		permWithPlus[0] = TopicToken{Value: "+", MatchType: SingleLevelMatch}
		copy(permWithPlus[1:], childPerm)
		perms = append(perms, permWithToken, permWithPlus)
	}
	return perms
}

func generateTokenWildcardPermsMultiLevel(tokens []TopicToken) [][]TopicToken {

	perms := make([][]TopicToken, len(tokens)+1)

	for i := len(tokens); i >= 0; i-- {
		ts := tokens[:i]
		p := make([]TopicToken, len(ts)+1)
		copy(p, ts)
		p[len(p)-1] = TopicToken{
			MatchType: MultiLevelMatch,
			Value:     "#",
		}
		perms[i] = p
	}

	return perms
}

func concatenateTopicToken(tokens []TopicToken) string {
	var strB strings.Builder
	strB.WriteString(tokens[0].Value)
	for _, t := range tokens[1:] {
		strB.WriteByte('/')
		strB.WriteString(t.Value)
	}
	return strB.String()
}

type genTopic struct {
	str    string
	tokens []TopicToken
}

func generateTokenWildcardPermutations(topic []byte) []genTopic {
	tokens, hasWildcards, err := ParseTopic(topic)
	if hasWildcards || err != nil {
		panic(fmt.Errorf("invalid topic for generateTokenWildcardPermutations:\n\t%s", string(topic)))
	}

	// num should be 2^t where t is the number of tokens
	singleLevelPs := generateTokenWildcardPermsSingleLevel(tokens)

	// num expected todo?
	multiLevelPs := make(map[string][]TopicToken)
	for _, sts := range singleLevelPs {
		mPs := generateTokenWildcardPermsMultiLevel(sts)
		for _, mts := range mPs {
			str := concatenateTopicToken(mts)
			multiLevelPs[str] = mts
		}
	}

	var genTopics []genTopic
	for _, ts := range singleLevelPs {
		genTopics = append(genTopics, genTopic{
			str:    concatenateTopicToken(ts),
			tokens: ts,
		})
	}
	for s, ts := range multiLevelPs {
		genTopics = append(genTopics, genTopic{
			str:    s,
			tokens: ts,
		})
	}

	// combine both and return
	return genTopics
}

func generateRandomTopics(n int) []string {
	topics := make([]string, 0, n)

	// credits: www.calhoun.io/creating-random-strings-in-go/
	genRandStr := func(length int) string {
		const charset = "abcdefghijklmnopqrstuvwxyz" +
			"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, length)
		for i := range b {
			b[i] = charset[rand.Intn(len(charset))]
		}
		return string(b)
	}

	genRandLevelStr := func() string {
		switch rand.Intn(4) {
		case 0:
			return "+"
		case 1:
			return "#"
		default:
			return genRandStr(rand.Intn(15))
		}
	}

	genRandTopic := func(maxLevels int) string {
		var strB strings.Builder
		// write first level
		levelStr := genRandLevelStr()
		if levelStr == "#" {
			return levelStr
		}
		strB.WriteString(levelStr)

		for i := 1; i < maxLevels; i++ {
			levelStr := genRandLevelStr()
			strB.WriteByte('/')
			strB.WriteString(levelStr)
			if levelStr == "#" {
				break
			}
		}
		return strB.String()
	}

	// ensure each topic is unique
	topicsSet := make(map[string]struct{}, n)

	for len(topics) != n {
		topic := genRandTopic(rand.Intn(15))
		if _, alreadyAdded := topicsSet[topic]; alreadyAdded {
			continue
		}
		topicsSet[topic] = struct{}{}
		topics = append(topics, topic)
	}
	return topics
}

func TestTopicMapInitFeedByTopic_ConcurrentInit(t *testing.T) {
	// config values
	const nTopics = 10 // number of topics to try to init
	nGoroutines := 10
	nAttemptsPerGoroutine := 1000

	type parsedTopic struct {
		str    string
		tokens []TopicToken
	}

	// set up everything
	var topics []parsedTopic
	for _, topic := range generateRandomTopics(nTopics) {
		tokens, _, err := ParseTopic([]byte(topic))
		require.NoError(t, err)
		topics = append(topics, parsedTopic{str: topic, tokens: tokens})
	}

	// number of concurrent go routines

	m := NewTopicMap()
	var wg sync.WaitGroup
	wg.Add(nGoroutines)
	var nInits [nTopics]int
	topicIniter := func() {
		for i := 0; i < nAttemptsPerGoroutine; i++ {
			// select random topic
			index := rand.Intn(nTopics)
			topic := topics[index]
			// init topic
			_, alreadyInit := m.InitFeedByTopic(topic.str, topic.tokens)
			if alreadyInit == false {
				nInits[index]++
			}
		}
		wg.Done()
	}
	for i := 0; i < nGoroutines; i++ {
		go topicIniter()
	}
	wg.Wait()
	for _, n := range nInits {
		assert.True(t, n <= 1)
	}
}

func TestTopicMapInitFeedByTopic_ReturnsSameFeed(t *testing.T) {
	// config values
	const nTopics = 10 // number of topics to try to init
	nGoroutines := 10
	nAttemptsPerGoroutine := 1000

	// set up everything
	type parsedTopic struct {
		str    string
		tokens []TopicToken
	}

	var topics []parsedTopic
	for _, topic := range generateRandomTopics(nTopics) {
		tokens, _, err := ParseTopic([]byte(topic))
		require.NoError(t, err)
		topics = append(topics, parsedTopic{str: topic, tokens: tokens})
	}
	m := NewTopicMap()
	var feeds [nTopics]*Feed
	for i, topic := range topics {
		feed, alreadyPresent := m.InitFeedByTopic(topic.str, topic.tokens)
		require.NotNil(t, feed)
		require.False(t, alreadyPresent)
		feeds[i] = feed
	}

	// get topics
	var wg sync.WaitGroup
	wg.Add(nGoroutines)
	topicGetter := func() {
		for i := 0; i < nAttemptsPerGoroutine; i++ {
			// select random topic
			index := rand.Intn(nTopics)
			topic := topics[index]
			// init topic
			feed, alreadyInit := m.InitFeedByTopic(topic.str, topic.tokens)
			if alreadyInit == false {
				t.Error("feed should already be init")
			}
			if feeds[index] != feed {
				t.Error("feed retrieved does not match expected feed for given topic")
			}
		}
		wg.Done()
	}
	for i := 0; i < nGoroutines; i++ {
		go topicGetter()
	}
	wg.Wait()

	// remove topics
	for i, topic := range topics {
		feed := m.RemoveFeedByTopic(topic.str, topic.tokens)
		require.NotNil(t, feed)
		require.Equal(t, feeds[i], feed)
	}
}

func TestTopicMap_GetFeedsThatMatchTopic_Basics(t *testing.T) {

	cases := []struct {
		publishTopics  []string
		shouldMatch    []string
		shouldNotMatch []string
	}{
		{
			publishTopics:  []string{"foo/bar/quz"},
			shouldMatch:    []string{"#", "foo/bar/quz", "foo/bar/+", "foo/bar/quz/#"},
			shouldNotMatch: []string{"+", "+/+", "foo/+/quz/+", "foo/+/quzz", "foo/bux/#", "ll/+/"},
		},
		{
			publishTopics:  []string{"sport/tennis/player1", "sport/tennis/player1/ranking", "sport/tennis/player1/score/wimbledon"},
			shouldMatch:    []string{"sport/tennis/player1/#"},
			shouldNotMatch: []string{"sport/tennis/player1/"},
		},
	}
	for _, cs := range cases {
		m := NewTopicMap()
		// add topics that should match
		for _, subscribeTopic := range cs.shouldMatch {
			tokens, _, err := ParseTopic([]byte(subscribeTopic))
			require.NoError(t, err)
			m.InitFeedByTopic(subscribeTopic, tokens)
		}

		// add topics that should not match
		for _, subscribeTopic := range cs.shouldNotMatch {
			tokens, _, err := ParseTopic([]byte(subscribeTopic))
			require.NoError(t, err)
			m.InitFeedByTopic(subscribeTopic, tokens)
		}

		// check that the publish topics match as required
		for _, publishTopic := range cs.publishTopics {
			tokens, _, err := ParseTopic([]byte(publishTopic))
			require.NoError(t, err)
			feeds := m.GetFeedsThatMatchTopic(tokens)
			require.Equal(t, len(cs.shouldMatch), len(feeds))
		}
	}
}

func TestTopicMap_GetFeedsThatMatchTopic_AllPermutations(t *testing.T) {
	topic := []byte("aaa/bbb/ccc/ddd/eee/fff")
	tokens, hasWildcards, err := ParseTopic(topic)
	require.NoError(t, err)
	require.False(t, hasWildcards)

	allWildcardPermutations := generateTokenWildcardPermutations(topic)

	ch := make(chan PublishEvent, len(allWildcardPermutations))
	m := NewTopicMap()
	for _, tp := range allWildcardPermutations {
		feed, alreadyPresent := m.InitFeedByTopic(tp.str, tp.tokens)
		if alreadyPresent == true {
			fmt.Println(tp.str, tp.tokens)
		}
		require.False(t, alreadyPresent)
		feed.Subscribe(ch)
	}

	feeds := m.GetFeedsThatMatchTopic(tokens)
	require.Equal(t, len(allWildcardPermutations), len(feeds))
	for _, f := range feeds {
		nSent := f.Publish(context.TODO(), nil)
		require.Equal(t, 1, nSent)
	}
	close(ch)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		received := 0
		for range ch {
			received++
		}
		require.Equal(t, len(allWildcardPermutations), received)
		wg.Done()
	}()

	wg.Wait()
}
