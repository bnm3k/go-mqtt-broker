package topics

import "errors"

// ErrInvalidTopicName is returned whenever a topic name or filter
// is invalid
var ErrInvalidTopicName = errors.New("invalid topic name/filter")

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

// MatchType indicates whether a match
// is an exact match(string), single-level or multi-level
type MatchType byte

// ExactMatch etc, see MatchType documentation entry
const (
	ExactMatch MatchType = iota
	SingleLevelMatch
	MultiLevelMatch
)

// TopicMatch holds a match for a topic filter
// plust whether it's a plain string or a wildcard
type TopicMatch struct {
	Match     string
	MatchType MatchType
}

// ParseTopicFilter parses a given bytes slice into a slice of topic filters
// each representing a topic level. For use mainly with Subscribe/Unsubscribe packets
// which might contain wildcards.
func ParseTopicFilter(b []byte) ([]TopicMatch, error) {
	if len(b) == 0 {
		// topic name must have 1 or more characters
		return nil, ErrInvalidTopicName
	}
	last := len(b) - 1
	var topics []TopicMatch
	var from, i int
	for i = 0; i < len(b); {
		c := b[i]
		// topic name should not contain  NUL character
		if c == 0 {
			return nil, ErrInvalidTopicName
		}

		// multilevel wildcard should only occur as last char
		if c == '#' {
			if i != last {
				return nil, ErrInvalidTopicName
			}
			topics = append(topics, TopicMatch{
				Match:     "#",
				MatchType: MultiLevelMatch,
			})
			return topics, nil
		}

		// single level wildcard should take up a whole level by itseld
		if c == '+' {
			if i != last && b[i+1] != '/' {
				return nil, ErrInvalidTopicName
			}
			if i != 0 && b[i-1] != '/' {
				return nil, ErrInvalidTopicName
			}
			topics = append(topics, TopicMatch{
				Match:     "+",
				MatchType: SingleLevelMatch,
			})
			// skip next sep
			i += 2
			from = i
			continue
		}

		if c == '/' {
			topics = append(topics, TopicMatch{
				Match:     string(b[from:i]),
				MatchType: ExactMatch,
			})
			from = i + 1
		}
		i++
	}
	if from < len(b) || b[last] == '/' {
		topics = append(topics, TopicMatch{
			Match:     string(b[from:i]),
			MatchType: ExactMatch,
		})
	}
	return topics, nil
}
