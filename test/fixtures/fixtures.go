package fixtures

import (
	// . "github.com/onsi/gomega"
	"github.com/provideapp/goldmine/test/matchers"
)

// Fixture interface
type Fixture interface{}

// FixtureMatcher unites fixture and the matcher
type FixtureMatcher struct {
	Fixture interface{}
	Matcher *matchers.MatcherCollection
}
