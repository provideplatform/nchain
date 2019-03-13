package matchers

import (
	"github.com/onsi/gomega/types"
)

// MatcherFunc is function receiving 0 args and returning GomegaMatcher result
type MatcherFunc func(opts ...interface{}) types.GomegaMatcher

// Matcher is the object combining name and MatcherFunc behavior
type Matcher struct {
	Name     *string
	Behavior MatcherFunc
	Options  *MatcherOptions
}

// MatcherOptions to hold matcher options
type MatcherOptions struct {
	ChannelPolling bool
	NATSPolling    bool
	NATSChannels   []*string
}

// NewMatcher to create Matcher
func NewMatcher(name *string, behavior MatcherFunc, opts map[string]interface{}) *Matcher {
	var chpoll bool
	chpoll = false
	var natspoll bool
	natspoll = false
	chs := []*string{}

	if opts["channelPolling"].(bool) {
		chpoll = true
	}
	if opts["natsPolling"].(bool) {
		natspoll = true
	}
	if v, ok := opts["natsChannels"]; ok {
		chs = v.([]*string)
	}

	return &Matcher{
		Name:     name,
		Behavior: behavior,
		Options: &MatcherOptions{
			ChannelPolling: chpoll,
			NATSPolling:    natspoll,
			NATSChannels:   chs,
		},
	}
}

// MatcherCollection is a set of Matcher
type MatcherCollection struct {
	//matchers []*Matcher
	matchers map[string]*Matcher
}

// MatchBehaviorFor is the function that matches behavior in tests
func (m *MatcherCollection) MatchBehaviorFor(name string, opts ...interface{}) types.GomegaMatcher {
	if b, ok := m.behaviorFor(name); ok {
		if len(opts) > 0 {
			return b(opts...)
		}
		return b()
	}
	return defaultBehavior()
}

// MatcherOptionsFor returns options for the matcher by its name
func (m *MatcherCollection) MatcherOptionsFor(name string) (*MatcherOptions, bool) {
	if b, ok := m.matchers[name]; ok {
		return b.Options, ok
	}
	return nil, false
}

// BehaviorFor is the function that matches behavior in tests
func (m *MatcherCollection) behaviorFor(name string) (MatcherFunc, bool) {
	if b, ok := m.matchers[name]; ok {
		return b.Behavior, ok
	}
	return nil, false
}

// AddBehavior to add Matcher element to inner matcher set
func (m *MatcherCollection) AddBehavior(name string, behavior MatcherFunc, opts map[string]interface{}) error {
	//m.matchers = append(m.matchers, &Matcher{Name: name, Behavior: behavior})
	// fmt.Printf("%v\n", name)
	if m.matchers == nil {
		m.matchers = map[string]*Matcher{}
	}
	m.matchers[name] = NewMatcher(&name, behavior, opts)
	// fmt.Printf("%v\n", reflect.ValueOf(m.matchers).MapKeys())

	return nil
}
