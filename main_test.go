package main

import (
	dbconf "github.com/kthomas/go-db-config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	// provideapp "github.com/provideapp/goldmine"
)

var _ = Describe("Main", func() {
	var n Network
	BeforeEach(func() {
		db := dbconf.DatabaseConnection()
		db.Delete(Network{})

		n = Network{
			ApplicationID: nil,
			UserID:        nil,
			Name:          ptrTo("Name ETH non-Cloneable Enabled"),
			Description:   ptrTo("Ethereum Network"),
			IsProduction:  ptrToBool(false),
			Cloneable:     ptrToBool(false),
			Enabled:       ptrToBool(true),
			ChainID:       nil,
			SidechainID:   nil,
			NetworkID:     nil,
			Config: marshalConfig(map[string]interface{}{
				"block_explorer_url":  "https://unicorn-explorer.provide.network", // required
				"chain":               "unicorn-v0",                               // required
				"chainspec_abi_url":   "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
				"chainspec_url":       "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json", // required If ethereum network
				"cloneable_cfg":       map[string]interface{}{},
				"engine_id":           "authorityRound", // required
				"is_ethereum_network": true,             // required for ETH
				"is_load_balanced":    true,             // implies network load balancer count > 0
				"json_rpc_url":        nil,
				"native_currency":     "PRVD", // required
				"network_id":          22,     // required
				"protocol_id":         "poa",  // required
				"websocket_url":       nil}),
			Stats: nil}

		// natsGuaranteeDelivery("network.create", t)
	})

	Describe("network", func() {
		Context("with config", func() {
			It("should be valid", func() {
				Expect(n.Create()).To(Equal(true))
			})
		})
	})

	Describe("network", func() {
		Context("without config", func() {
			It("should be invalid", func() {
				n.Config = nil
				Expect(n.Create()).To(Equal(false))
			})
		})
	})

	Describe("panic", func() {
		It("panics in a goroutine", func(done Done) {
			go func() {
				defer GinkgoRecover()

				Ω(n.Create()).Should(BeTrue())

				close(done)
			}()
		})
	})
})
