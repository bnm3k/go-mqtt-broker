package topics

// level represents a single level for a given topic
type level struct {
	token    string
	children map[string]*level
	Value    interface{}
}

// TopicMap holds a map of topics to values
type TopicMap struct {
	root *level
}

// TraverseFn allows callers of traverse to iterate over topics that match
// a give filter and access the relevant values
type TraverseFn func(interface{}) bool

// NewTopicMap returns a new instance of a topic map
func NewTopicMap() *TopicMap {
	return &TopicMap{
		root: &level{
			token:    "::root::",
			children: make(map[string]*level),
		},
	}
}

// Insert inserts a given topic + value
func (m *TopicMap) Insert(topicName []byte, val interface{}) error {
	tokens, err := ParseTopicName(topicName)
	if err != nil {
		return err
	}
	curr := m.root
	for _, token := range tokens {
		next, present := curr.children[token]
		if !present {
			next = &level{
				token:    token,
				children: make(map[string]*level),
			}
			curr.children[token] = next
		}
		curr = next
	}
	curr.Value = val
	return nil
}

// Get ...
func (m *TopicMap) Get(topicName []byte) (val interface{}, err error) {
	topic, err := ParseTopicName(topicName)
	if err != nil {
		return err, nil
	}
	last := len(topic) - 1
	curr := m.root
	for i := 0; i <= last; i++ {
		var present bool
		curr, present = curr.children[topic[i]]
		if !present {
			return
		}
	}
	val = curr.Value
	return
}

// Traverse ...
func (m *TopicMap) Traverse(topic []byte, fn TraverseFn) (err error) {
	topicTokens, hasWildcard, err := ParseTopic(topic)
	if err != nil {
		return err
	}
	// if no wildcard, simply do an exact match, skip extra work
	if hasWildcard == false {
		val, _ := m.Get(topic)
		fn(val)
		return
	}
	last := len(topicTokens) - 1
	var matches []*level
	// get first level matches, there must be at least 1 token
	firstToken := topicTokens[0]
	switch firstToken.MatchType {
	case ExactMatch:
		if c, ok := m.root.children[firstToken.Value]; ok {
			matches = append(matches, c)
		}
	case SingleLevelMatch:
		for _, c := range m.root.children {
			matches = append(matches, c)
		}
	case MultiLevelMatch:
		for _, c := range m.root.children {
			traverseMultilLevel(c, fn)
		}
		return
	}

	// continue getting matches
	for i := 1; i <= last; i++ {
		var token TopicMatch = topicTokens[i]
		var filtered []*level

		switch token.MatchType {
		case ExactMatch:
			for _, match := range matches {
				if c, ok := match.children[token.Value]; ok {
					filtered = append(filtered, c)
				}
			}
		case SingleLevelMatch:
			for _, match := range matches {
				for _, c := range match.children {
					filtered = append(filtered, c)
				}
			}
		case MultiLevelMatch:
			for _, c := range matches {
				traverseMultilLevel(c, fn)
			}
			return
		}

		matches = filtered
	}

	// traverse
	for _, l := range matches {
		if l.Value != nil {
			shouldContinue := fn(l.Value)
			if !shouldContinue {
				return
			}
		}
		// for _, c := range l.children {
		// 	shouldContinue = fn(c.Value)
		// 	if !shouldContinue {
		// 		return
		// 	}
		// }
	}
	return
}

func traverseMultilLevel(level *level, fn TraverseFn) {
	if level.Value != nil {
		shouldContinue := fn(level.Value)
		if !shouldContinue {
			return
		}
	}
	for _, childLevel := range level.children {
		traverseMultilLevel(childLevel, fn)
	}
}
