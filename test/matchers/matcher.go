package matchers

import (
	"github.com/onsi/gomega/types"
)

// MatcherFunc is function receiving 0 args and returning GomegaMatcher result
type MatcherFunc func() types.GomegaMatcher

// Matcher is the object combining name and MatcherFunc behavior
type Matcher struct {
	Name     *string
	Behavior MatcherFunc
}

// MatcherCollection is a set of Matcher
type MatcherCollection struct {
	//matchers []*Matcher
	matchers map[string]*Matcher
}

// MatchBehaviorFor is the function that matches behavior in tests
func (m *MatcherCollection) MatchBehaviorFor(name string) types.GomegaMatcher {
	if b, ok := m.behaviorFor(name); ok {
		return b()
	}
	return defaultBehavior()
}

// BehaviorFor is the function that matches behavior in tests
func (m *MatcherCollection) behaviorFor(name string) (MatcherFunc, bool) {
	if b, ok := m.matchers[name]; ok {
		return b.Behavior, ok
	}
	return nil, false
}

// AddBehavior to add Matcher element to inner matcher set
func (m *MatcherCollection) AddBehavior(name string, behavior MatcherFunc) error {
	//m.matchers = append(m.matchers, &Matcher{Name: name, Behavior: behavior})
	if m.matchers == nil {
		m.matchers = map[string]*Matcher{}
	}
	m.matchers[name] = &Matcher{Name: &name, Behavior: behavior}
	return nil
}
