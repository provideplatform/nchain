package main

import (
	"fmt"

	dbconf "github.com/kthomas/go-db-config"
	stan "github.com/nats-io/go-nats-streaming"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	//matchers "github.com/provideapp/goldmine/matchers/network_matchers"
	//. "github.com/onsi/gomega/types"
	// provideapp "github.com/provideapp/goldmine"
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

var _ = Describe("Main", func() {
	var n Network
	var ch chan *stan.Msg
	// var chStr chan string
	var natsConn stan.Conn
	var natsSub stan.Subscription
	var err error

	BeforeEach(func() {

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
				"block_explorer_url": "https://unicorn-explorer.provide.network", // required
				"chain":              "unicorn-v0",                               // required
				"chainspec_abi_url":  "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
				"chainspec_url":      "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json", // required If ethereum network
				"cloneable_cfg": map[string]interface{}{
					"_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security
				"engine_id":           "authorityRound", // required
				"is_ethereum_network": true,             // required for ETH
				"is_load_balanced":    true,             // implies network load balancer count > 0
				"json_rpc_url":        nil,
				"native_currency":     "PRVD", // required
				"network_id":          22,     // required
				"protocol_id":         "poa",  // required
				"websocket_url":       nil}),
			Stats: nil}

		sub := "network.create"
		natsConn = getNatsStreamingConnection()
		ch = make(chan *stan.Msg, 1)
		natsSub, err = natsConn.QueueSubscribe(sub, sub, func(msg *stan.Msg) {
			Log.Debugf("subject: " + msg.MsgProto.Subject)
			data := string(msg.MsgProto.Data)
			Log.Debugf("data: " + data)
			Log.Debugf("data: " + string(msg.MsgProto.Data))
			ch <- msg
			// chStr <- string(msg.MsgProto.Data)

			//wg.Done()
		})
		if err != nil {
			Log.Debugf("conn failure")
		}

		//natsGuaranteeDelivery("network.create")
	})

	AfterEach(func() {
		db := dbconf.DatabaseConnection()
		db.Delete(Network{})

		natsSub.Unsubscribe()
	})

	Describe("network.Create()", func() {
		Context("production", func() {})

		Context("ETH", func() {

			Context("non production", func() {
				// BeforeEach(func() {
				// 	n.IsProduction = ptrToBool(false)
				// })

				Context("cloneable", func() {
					BeforeEach(func() {
						n.Cloneable = ptrToBool(true)
					})

					Context("enabled", func() {
						// n.Enabled set by default

						Context("with config", func() {
							// n.Config set by default
							It("should be valid", func() {
								Expect(n.Validate()).To(BeTrue())
							})
							It("should be created", func() {
								Expect(n.Create()).To(NetworkCreateMatcher("network.create"))
							})
						})
						Context("with nil config", func() {
							BeforeEach(func() {
								n.Config = nil
							})
							It("should be invalid", func() {
								Expect(n.Validate()).To(BeFalse())
							})
							It("should not be created", func() {
								Expect(n.Create()).To(BeFalse())
							})
						})
					})
					Context("disabled", func() {
						BeforeEach(func() {
							n.Enabled = ptrToBool(false)
						})

						Context("with config", func() {
							It("should be valid", func() {
								Expect(n.Validate()).To(BeTrue())
							})
						})
						Context("with nil config", func() {
							BeforeEach(func() {
								n.Config = nil
							})
							It("should be invalid", func() {
								Expect(n.Validate()).To(BeFalse())
							})
							It("should not be created", func() {
								Expect(n.Create()).To(BeFalse())
							})
						})
						Context("with empty config", func() {
							BeforeEach(func() {
								n.Config = marshalConfig(map[string]interface{}{})
							})
							It("should be invalid", func() {
								Expect(n.Validate()).To(BeFalse())
							})
							It("should not be created", func() {
								Expect(n.Create()).To(BeFalse())
							})
						})
					})
				})
				Context("non cloneable", func() {
					// non clonable by default

					Context("enabled", func() {
						// n.Enabled set by default

						Context("with config", func() {
							// n.Config set by default

							It("should be valid", func() {
								Expect(n.Validate()).To(BeTrue())
							})
							It("should create successfully and send message to NATS", func() {
								// Log.Debugf("%v", n)
								Expect(n.Create()).To(BeTrue())

								// Eventually(chStr).Should(Receive(Equal(n.ID.String())))

								Eventually(ch).Should(Receive(PointTo(
									MatchFields(IgnoreExtras, Fields{
										"MsgProto": MatchFields(IgnoreExtras, Fields{
											"Subject": Equal("network.create"),
											"Data":    BeEquivalentTo([]byte(n.ID.String()))}),
									}),
								)))
							})
							It("should have nil NetworkID", func() {
								Expect(n.NetworkID).To(BeNil())
							})
							It("should have ChainID", func() {
								n.Create()
								Expect(n.ChainID).NotTo(BeNil()) // (PointTo(Equal("0x5c77fad4"))) it always changes
							})
							It("should not have errors", func() {
								Expect(n.Errors).To(BeEmpty())
							})
							It("should be ETH network", func() {
								Expect(n.isEthereumNetwork()).To(BeTrue(), "it's ETH network")
								Expect(n.isBcoinNetwork()).To(BeFalse())
								Expect(n.isHandshakeNetwork()).To(BeFalse())
								Expect(n.isLcoinNetwork()).To(BeFalse())
								Expect(n.isQuorumNetwork()).To(BeFalse())
							})

							It("should have config parsed correctly", func() {
								parsedConfig := n.ParseConfig()
								Expect(parsedConfig).To(HaveKey("block_explorer_url"))
								Expect(parsedConfig).To(HaveKey("chain"))
								Expect(parsedConfig).To(HaveKey("chainspec_url"))
								Expect(parsedConfig).To(HaveKey("chainspec_abi_url"))
								Expect(parsedConfig).To(HaveKey("cloneable_cfg"))
								Expect(parsedConfig).To(HaveKey("engine_id"))
								Expect(parsedConfig).To(HaveKey("is_ethereum_network"))
								Expect(parsedConfig).To(HaveKey("is_load_balanced"))
								Expect(parsedConfig).To(HaveKey("json_rpc_url"))
								Expect(parsedConfig).To(HaveKey("native_currency"))
								Expect(parsedConfig).To(HaveKey("network_id"))
								Expect(parsedConfig).To(HaveKey("protocol_id"))
								Expect(parsedConfig).To(HaveKey("websocket_url"))
							})
						})
						Context("with nil config", func() {
							BeforeEach(func() {
								n.Config = nil
							})
							It("should be invalid", func() {
								Expect(n.Create()).To(Equal(false))

								Expect(n.Errors).To(HaveLen(1))
								Expect(n.Errors[0]).To(PointTo(MatchAllFields(Fields{
									"Message": PointTo(BeEquivalentTo("Config value should be present")),
									"Status":  PointTo(BeEquivalentTo(10)),
								})))

								// id := func(element *provide.Error) string {
								// 	return *(element.Message)
								// }
								// Expect(n.Errors).To(MatchAllElements(id, Elements{
								// 	"0": PointTo(MatchAllFields(Fields{
								// 		"Message": BeEquivalentTo("Config value should be present"),
								// 		"Status":  BeEquivalentTo(10),
								// 	})),
								// }))

							})
						})
					})
					Context("disabled", func() {
						BeforeEach(func() {
							n.Enabled = ptrToBool(false)
						})
						Context("with config", func() {
							It("should be valid", func() {
								Expect(n.Validate()).To(BeTrue())
							})
							It("should be created", func() {
								Expect(n.Create()).To(BeTrue())
							})
						})
						Context("with nil config", func() {
							BeforeEach(func() {
								n.Config = nil
							})
							It("should be invalid", func() {
								Expect(n.Validate()).To(BeFalse())
							})
							It("should not be created", func() {
								Expect(n.Create()).To(BeFalse())
							})
						})
						Context("with empty config", func() {
							BeforeEach(func() {
								n.Config = marshalConfig(map[string]interface{}{})
							})
							It("should be invalid", func() {
								Expect(n.Validate()).To(BeFalse())
							})
							It("should not be created", func() {
								Expect(n.Create()).To(BeFalse())
							})
						})
					})
				})

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
