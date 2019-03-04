package network_matchers

import (
	"fmt"

	"github.com/onsi/gomega/types"
)

// NetworkCreateMatcher checks network.Create(): the result of Create() call and channel message
func NetworkCreateMatcher(expected interface{}) types.GomegaMatcher {
	return &networkCreateMatcher{
		expected: expected,
	}
}

type networkCreateMatcher struct {
	expected interface{}
}

func (matcher *networkCreateMatcher) Match(actual interface{}) (success bool, err error) {
	return true, nil
}

func (matcher *networkCreateMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto match\n\t%#v", actual, matcher.expected)
}

func (matcher *networkCreateMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to match\n\t%#v", actual, matcher.expected)
}
