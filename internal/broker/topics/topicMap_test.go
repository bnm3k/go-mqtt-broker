package topics

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTopicMap(t *testing.T) {
	topic := []byte("/")
	tm := NewTopicMap()
	err := tm.Insert(topic, 100)
	require.NoError(t, err)
	val, err := tm.Get(topic)
	require.NoError(t, err)
	require.Equal(t, 100, val)

}

func TestTopicMapTraverse(t *testing.T) {
	tm := NewTopicMap()

	tm.Insert([]byte("a"), "a")
	tm.Insert([]byte("a/q/w/e/r/t/y/u"), "a/q/w/e/r/t/y/u")
	tm.Insert([]byte("n/m"), "n/m")
	tm.Insert([]byte("a/b"), "a/b")
	tm.Insert([]byte("a/x/a"), "a/x/a")
	tm.Insert([]byte("a/x/b"), "a/x/b")
	tm.Insert([]byte("a/x/c"), "a/x/c")
	tm.Insert([]byte("a/y/c"), "a/y/c")
	tm.Insert([]byte("a/z/c"), "a/z/c")
	tm.Insert([]byte("foo/bar/c"), "foo/bar/c")

	// tm.Insert([]byte("sport/tennis/player1"), "sport/tennis/player1")
	// tm.Insert([]byte("sport/tennis/player1/ranking"), "sport/tennis/player1/ranking")
	// tm.Insert([]byte("sport/tennis/player1/score/wimbledon"), "sport/tennis/player1/score/wimbledon")

	fmt.Printf("\n-------------------\n")
	err := tm.Traverse([]byte("a/+/#"), func(val interface{}) bool {
		fmt.Printf("%v\n", val)
		return true
	})
	fmt.Printf("-------------------\n\n")
	require.NoError(t, err)
}

func TestTopicMapReverse(t *testing.T) {
	fmt.Print("\n-------------------\n")
	// var err error
	checkErr := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m := newWilcardTopicMap()

	// // should not match
	checkErr(t, m.InsertWilcardTopic([]byte("+"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("+/+"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("foo/+/quz/+"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("foo/+/quzz"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("foo/bux/#"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("ll/+/"), "1"))

	// // should match
	checkErr(t, m.InsertWilcardTopic([]byte("#"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("#"), "2"))
	checkErr(t, m.InsertWilcardTopic([]byte("foo/#"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("foo/bar/#"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("foo/bar/+"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("+/+/+"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("foo/+/+"), "1"))
	checkErr(t, m.InsertWilcardTopic([]byte("foo/+/quz"), "1"))

	m.RemoveByID("2")
	checkErr(t, m.TraverseMatchingWildcardTopics([]byte("foo/bar/quz")))

	checkErr(t, m.InsertWilcardTopic([]byte("sport/tennis/player1/#"), "1"))

	checkErr(t, m.TraverseMatchingWildcardTopics([]byte("sport/tennis/player1")))
	checkErr(t, m.TraverseMatchingWildcardTopics([]byte("sport/tennis/player1/ranking")))
	checkErr(t, m.TraverseMatchingWildcardTopics([]byte("sport/tennis/player1/score/wimbledon")))

	fmt.Print("-------------------\n\n")
}
