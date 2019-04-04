package networkfixtures

import (
	"fmt"

	"github.com/provideapp/goldmine/test/fixtures"
)

// NetworkFixtureDispatcher to hold fixtures with matchers
type NetworkFixtureDispatcher struct {
	generator       *NetworkFixtureGenerator
	fixtureMatchers []*fixtures.FixtureMatcher
}

// NewNetworkFixtureDispatcher returns NetworkFixtureDispatcher with provided generator
func NewNetworkFixtureDispatcher(generator *NetworkFixtureGenerator) (nfd *NetworkFixtureDispatcher) {
	// fmt.Printf("%v", generator)
	networks := Networks()
	nfd = &NetworkFixtureDispatcher{
		generator: generator,
	}

	for _, n := range networks {
		fields := n.Fixture.(*NetworkFixture).Fields
		mc := n.Matcher
		fieldSets, possibleFieldSets := generator.Select(fields)

		fmt.Printf("\nFound %v fixtures for filter '%v'\n", len(fieldSets), *n.Fixture.(*NetworkFixture).Name)
		if len(possibleFieldSets) > 0 {
			fmt.Printf("Possible field sets: \n")
			for _, fs := range possibleFieldSets {
				fmt.Printf("    '%v'\n", *fs.Name)
			}
		}

		if len(fieldSets) == 0 {
			fmt.Printf("NO FIXTURE: %v\n", *n.Fixture.(*NetworkFixture).Name)
		}
		for _, f := range fieldSets {
			fm := &fixtures.FixtureMatcher{
				Fixture: &NetworkFixture{
					Fields: f,
					Name:   f.Name,
				},
				Matcher: mc,
			}
			fmt.Printf("Adding '%v' fixture for filter '%v'\n", *f.Name, *n.Fixture.(*NetworkFixture).Name)
			nfd.fixtureMatchers = append(nfd.fixtureMatchers, fm)

		}
	}
	return
}

// NotCovered method returns the list of fixtures generated but not covered with test matchers.
func (dispatcher *NetworkFixtureDispatcher) NotCovered() []*NetworkFixture {
	// (generator.networks - dispatcher.fixtureMatchers.collect(&:Fixture)).uniq
	fixturePtrs := []*NetworkFixture{}
	allNetworks := dispatcher.generator.All()
	fmt.Printf("all: %v\n", len(allNetworks))
	fmt.Printf("matchers: %v\n", len(dispatcher.fixtureMatchers))
	var networkUsed bool
	for _, n := range allNetworks {
		networkUsed = false
		for _, fm := range dispatcher.fixtureMatchers {
			fixture := fm.Fixture
			fields := fixture.(*NetworkFixture).Fields
			// fieldSetPtrs = append(fieldSetPtrs)
			if n == fields {
				networkUsed = true
			}
		}
		if !networkUsed {
			fixturePtrs = append(fixturePtrs, &NetworkFixture{
				Fields: n,
				Name:   n.Name,
			})
		}
	}
	return fixturePtrs
}

// Networks method returns dispatcher fixtures with matchers
func (dispatcher *NetworkFixtureDispatcher) Networks() []*fixtures.FixtureMatcher {
	return dispatcher.fixtureMatchers
}
