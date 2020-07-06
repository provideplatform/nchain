package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNChainAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NChain API Suite")
}

var _ = Describe("Main", func() {

})
