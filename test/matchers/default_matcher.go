package matchers

import (
	"fmt"

	"github.com/onsi/gomega/types"
)

type defaultMatcher struct {
	expected interface{}
}

func defaultBehavior() types.GomegaMatcher {
	return &defaultMatcher{expected: nil}
}

func (matcher *defaultMatcher) Match(actual interface{}) (success bool, err error) {
	return false, fmt.Errorf("no matcher provided")
}

func (matcher *defaultMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto contain the matcher behavior", actual)
}

func (matcher *defaultMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to contain the matcher behavior", actual)
}
