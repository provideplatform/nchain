package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNChainStatsdaemon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NChain statsdaemon Suite")
}

var _ = Describe("Main", func() {

})
