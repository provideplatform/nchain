package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNChainConsumer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NChain Consumer Suite")
}

var _ = Describe("Main", func() {

})
