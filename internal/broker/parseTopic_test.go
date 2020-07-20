package broker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTopicFilter(t *testing.T) {
	cases := []struct {
		description    string
		topicName      []byte
		expectedLevels []MatchType
		hasWildCard    bool
		errShouldOccur bool
	}{
		{
			"should be able to parse standard  topic filter",
			[]byte("foo/bar"),
			[]MatchType{ExactMatch, ExactMatch},
			false,
			false,
		},

		{
			"should be able to parse a single string with no separator",
			[]byte("hello world"),
			[]MatchType{ExactMatch},
			false,
			false,
		},

		{
			"should be able to parse '/' at start",
			[]byte("/bar/buz"),
			[]MatchType{ExactMatch, ExactMatch, ExactMatch},
			false,
			false,
		},
		{
			"should be able to parse '/' at end",
			[]byte("foo/bar/buz/"),
			[]MatchType{ExactMatch, ExactMatch, ExactMatch, ExactMatch},
			false,
			false,
		},
		{
			"should be able to parse a single '/'",
			[]byte("/"),
			[]MatchType{ExactMatch, ExactMatch},
			false,
			false,
		},
		{
			"should be able to parse consecutive '///' 1",
			[]byte("///"),
			[]MatchType{ExactMatch, ExactMatch, ExactMatch, ExactMatch},
			false,
			false,
		},
		{
			"should be able to parse consecutive '///' 2",
			[]byte("a///b"),
			[]MatchType{ExactMatch, ExactMatch, ExactMatch, ExactMatch},
			false,
			false,
		},
		{
			"should be able to parse consecutive '///' 3",
			[]byte("a/b/////"),
			[]MatchType{ExactMatch, ExactMatch, ExactMatch, ExactMatch, ExactMatch, ExactMatch, ExactMatch},
			false,
			false,
		},
		{
			"handles single '+'",
			[]byte("+"),
			[]MatchType{SingleLevelMatch},
			true,
			false,
		},
		{
			"handles '+' at start 1",
			[]byte("+/"),
			[]MatchType{SingleLevelMatch, ExactMatch},
			true,
			false,
		},
		{
			"handles '+' at start 2",
			[]byte("+/foo/bar"),
			[]MatchType{SingleLevelMatch, ExactMatch, ExactMatch},
			true,
			false,
		},
		{
			"handles '+' at end 1",
			[]byte("/+"),
			[]MatchType{ExactMatch, SingleLevelMatch},
			true,
			false,
		},
		{
			"handles '+' at end 2",
			[]byte("foo/bar/+"),
			[]MatchType{ExactMatch, ExactMatch, SingleLevelMatch},
			true,
			false,
		},
		{
			"handles '+' in middle",
			[]byte("foo/+/bar"),
			[]MatchType{ExactMatch, SingleLevelMatch, ExactMatch},
			true,
			false,
		},
		{
			"handles multiple '+' ",
			[]byte("foo/+/bar/+/+/buz"),
			[]MatchType{ExactMatch, SingleLevelMatch, ExactMatch, SingleLevelMatch, SingleLevelMatch, ExactMatch},
			true,
			false,
		},
		{
			"handles single '#'",
			[]byte("#"),
			[]MatchType{MultiLevelMatch},
			true,
			false,
		},
		{
			"handles '#' 1",
			[]byte("/#"),
			[]MatchType{ExactMatch, MultiLevelMatch},
			true,
			false,
		},
		{
			"handles '#' 2",
			[]byte("foo/bar/baz/#"),
			[]MatchType{ExactMatch, ExactMatch, ExactMatch, MultiLevelMatch},
			true,
			false,
		},
		{
			"handles '#' 3",
			[]byte("foo/+/baz/#"),
			[]MatchType{ExactMatch, SingleLevelMatch, ExactMatch, MultiLevelMatch},
			true,
			false,
		},
		{
			"error when zero length topic provided",
			[]byte(""),
			nil,
			false,
			true,
		},
		{
			"error when '#' not at end 1",
			[]byte("foo/#/bar"),
			nil,
			false,
			true,
		},
		{
			"error when '#' not at end 2",
			[]byte("#/"),
			nil,
			false,
			true,
		},
		{
			"error when '+' consecutive",
			[]byte("foo/++/bar"),
			nil,
			false,
			true,
		},
		{
			"error when wildcard contained within a topic name 1",
			[]byte("foo/bar+/buz"),
			nil,
			false,
			true,
		},

		{
			"error when wildcard contained within a topic name 2",
			[]byte("foo/bar#/buz"),
			nil,
			false,
			true,
		},
		{
			"error when NUL character within topic name",
			[]byte{97, 97, 97, 47, 98, 0, 98, 47, 99, 99, 99},
			nil,
			false,
			true,
		},
	}
	for _, cs := range cases {
		t.Run(cs.description, func(t *testing.T) {
			topic, hasWildcard, err := ParseTopic(cs.topicName)
			if cs.errShouldOccur {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, cs.hasWildCard, hasWildcard)
				require.Equal(t, len(cs.expectedLevels), len(topic))
				for i := 0; i < len(topic); i++ {
					require.Equal(t, cs.expectedLevels[i], topic[i].MatchType)
				}
			}
		})
	}
}

func TestParseTopicName(t *testing.T) {
	cases := []struct {
		description    string
		topicName      []byte
		expectedLevels int
		errShouldOccur bool
	}{
		{
			"should be able to parse standard  topic name",
			[]byte("foo/bar"),
			2,
			false,
		},
		{
			"should be able to parse a single string with no separator",
			[]byte("hello world"),
			1,
			false,
		},
		{
			"should be able to parse '/' at start",
			[]byte("/bar/buz"),
			3,
			false,
		},
		{
			"should be able to parse '/' at end",
			[]byte("foo/bar/buz/"),
			4,
			false,
		},
		{
			"should be able to parse a single '/'",
			[]byte("/"),
			2,
			false,
		},
		{
			"should be able to parse consecutive '///'",
			[]byte("///"),
			4,
			false,
		},
		{
			"should be able to parse consecutive '///'",
			[]byte("a///b"),
			4,
			false,
		},
		{
			"should be able to parse consecutive '///'",
			[]byte("a/b/////"),
			7,
			false,
		},
		{
			"error when zero length topic provided",
			[]byte(""),
			-1,
			true,
		},
		{
			"error when wildcard '+' provided",
			[]byte("foo/+/bar"),
			-1,
			true,
		},
		{
			"error when wildcard '#' provided",
			[]byte("foo/bar/#"),
			-1,
			true,
		},
		{
			"error when wildcard contained within a topic name",
			[]byte("foo/bar+/buz"),
			-1,
			true,
		},
		{
			"error when NUL character within topic name",
			[]byte{97, 97, 97, 47, 98, 0, 98, 47, 99, 99, 99},
			-1,
			true,
		},
	}
	for _, cs := range cases {
		t.Run(cs.description, func(t *testing.T) {
			topic, err := ParseTopicName(cs.topicName)
			if cs.errShouldOccur {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, cs.expectedLevels, len(topic))
			}
		})
	}
}
