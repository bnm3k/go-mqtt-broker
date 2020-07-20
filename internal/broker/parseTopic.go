package broker

import "errors"

// ErrInvalidTopicName is returned whenever a topic name or filter
// is invalid
var ErrInvalidTopicName = errors.New("invalid topic name/filter")

// MatchType indicates whether a match
// is an exact match(string), single-level or multi-level
type MatchType byte

// ExactMatch etc, see MatchType documentation entry
const (
	ExactMatch MatchType = iota
	SingleLevelMatch
	MultiLevelMatch
)

// TopicToken holds a match for a topic filter
// plust whether it's a plain string or a wildcard
type TopicToken struct {
	Value     string
	MatchType MatchType
}

// ParseTopic parses a given bytes slice into a slice of topic filters
// each representing a topic level. For use mainly with Subscribe/Unsubscribe packets
// which might contain wildcards.
func ParseTopic(b []byte) (tokens []TopicToken, hasWildcard bool, err error) {
	if len(b) == 0 {
		// topic name must have 1 or more characters
		return nil, false, ErrInvalidTopicName
	}
	last := len(b) - 1
	var from, i int
	for i = 0; i < len(b); {
		c := b[i]
		// topic name should not contain  NUL character
		if c == 0 {
			return nil, false, ErrInvalidTopicName
		}

		// multilevel wildcard should only occur as last char
		if c == '#' {
			if i != last {
				return nil, false, ErrInvalidTopicName
			}
			tokens = append(tokens, TopicToken{
				Value:     "#",
				MatchType: MultiLevelMatch,
			})
			return tokens, true, nil
		}

		// single level wildcard should take up a whole level by itseld
		if c == '+' {
			hasWildcard = true
			if i != last && b[i+1] != '/' {
				return nil, false, ErrInvalidTopicName
			}
			if i != 0 && b[i-1] != '/' {
				return nil, false, ErrInvalidTopicName
			}
			tokens = append(tokens, TopicToken{
				Value:     "+",
				MatchType: SingleLevelMatch,
			})
			// skip next sep
			i += 2
			from = i
			continue
		}

		if c == '/' {
			tokens = append(tokens, TopicToken{
				Value:     string(b[from:i]),
				MatchType: ExactMatch,
			})
			from = i + 1
		}
		i++
	}
	if from < len(b) || b[last] == '/' {
		tokens = append(tokens, TopicToken{
			Value:     string(b[from:i]),
			MatchType: ExactMatch,
		})
	}
	return tokens, hasWildcard, nil
}

// ParseTopicName parses a given bytes slice into a slice of strings
// each representing a topic level. For use mainly with Publish packets
// which should not contain wildcards.
func ParseTopicName(b []byte) ([]string, error) {
	if len(b) == 0 {
		// topic name must have 1 or more characters
		return nil, ErrInvalidTopicName
	}
	var topics []string
	var from, i int
	for i = 0; i < len(b); i++ {
		c := b[i]
		if c == '#' || c == '+' || c == 0 {
			// topic name should not contain wildcards or NUL character
			return nil, ErrInvalidTopicName
		}
		if c == '/' {
			topics = append(topics, string(b[from:i]))
			from = i + 1
		}

	}
	if from < len(b) || b[len(b)-1] == '/' {
		topics = append(topics, string(b[from:]))
	}
	return topics, nil
}
