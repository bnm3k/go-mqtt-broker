package topics

import (
	"fmt"
	"strings"
)

// wNode holds a level for a wildcard match
type wNode struct {
	children map[string]*wNode
	items    map[string]interface{}
}

type wildcardTopicMap struct {
	root *wNode
}

// newWilcardTopicMap returns an instance of a topic map
// for holding topics with wildcards
func newWilcardTopicMap() wildcardTopicMap {
	return wildcardTopicMap{
		root: &wNode{
			children: make(map[string]*wNode),
			items:    make(map[string]interface{}),
		},
	}
}

// InsertWilcardTopic. Topic should have at least 1 token
func (m wildcardTopicMap) InsertWilcardTopic(topic []byte, id string) (err error) {
	// check that given topic has wildcards present
	tokens, hasWildcard, err := ParseTopic(topic)
	if err != nil {
		return
	}
	if hasWildcard == false {
		return ErrInvalidTopicName
	}
	curr := m.root
	for _, token := range tokens {
		next, present := curr.children[token.Value]
		if !present {
			next = &wNode{
				children: make(map[string]*wNode),
				items:    make(map[string]interface{}),
			}
			curr.children[token.Value] = next
		}
		curr = next
	}
	curr.items[id] = nil
	return
}

// RemoveByID ...
func (m wildcardTopicMap) RemoveByID(id string) {
	removeByID(m.root, id)
}

func removeByID(n *wNode, id string) {
	delete(n.items, id)
	for _, c := range n.children {
		removeByID(c, id)
	}
}

type onMatchFn func([]string, map[string]interface{})

// traverseMatchingWildcardTopics.The given topic should be an exact topic match, ie,
// it should not have any wildcards. For use mainly when a publish packet
// arrives and one needs to check whether a subscriber qualifies to receive
// it on the basis of a topic they supplied earlier which contained wildcards
func (m wildcardTopicMap) TraverseMatchingWildcardTopics(topic []byte) (err error) {
	// tokens should not contain any wildcards
	tokens, err := ParseTopicName(topic)
	if err != nil {
		return
	}

	// to hold topic matching wildcard topic
	var strB strings.Builder
	strB.Grow(len(topic))
	i := 1

	handleMatch := func(ms []string, items map[string]interface{}) {
		// avoid doing unnecessary work
		if len(items) == 0 {
			return
		}
		strB.WriteString(ms[0])
		for _, m := range ms[1:] {
			strB.WriteByte('/')
			strB.WriteString(m)
		}
		wildcardTopic := strB.String()
		for id := range items {
			s := fmt.Sprintf("\n%2d. %s -> (%s), %+v\n", i, string(topic), wildcardTopic, id)
			fmt.Println(s)
			i++
		}
		strB.Reset()
	}

	// find matching wildcard topics
	fs := make([]string, 0, len(tokens))
	findMatchingWildcardTopics(fs, tokens, m.root, handleMatch)

	fmt.Println()
	return
}

func findMatchingWildcardTopics(fs []string, tokens []string, curr *wNode, handleMatch onMatchFn) {
	// first check single level matches
	matches := [2]string{"+", tokens[0]}
	for _, m := range matches {
		if v, ok := curr.children[m]; ok {
			nextFs := append(fs, m)
			if len(tokens) == 1 { // last token
				handleMatch(nextFs, v.items)
				// # matches parent level too
				if v, ok := v.children["#"]; ok {
					handleMatch(append(nextFs, "#"), v.items)
				}
			} else { // more remaining tokens
				findMatchingWildcardTopics(nextFs, tokens[1:], v, handleMatch)
			}
		}
	}

	// check multi-level matches
	if v, ok := curr.children["#"]; ok {
		handleMatch(append(fs, "#"), v.items)
	}
}
