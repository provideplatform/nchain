package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGoldmineConsumer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Goldmine Consumer Suite")
}

var _ = Describe("Main", func() {

})
