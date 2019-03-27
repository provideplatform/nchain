package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func ptrTo(s string) *string {
	return &s
}

func ptrToBool(b bool) *bool {
	return &b
}

func TestGoldmine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Goldmine Suite")
}

var _ = Describe("Main", func() {

})
