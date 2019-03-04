package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	dbconf "github.com/kthomas/go-db-config"
	stan "github.com/nats-io/go-nats-streaming"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "github.com/provideapp/goldmine/matchers"
	//. "github.com/onsi/gomega/types"
	// provideapp "github.com/provideapp/goldmine"
)

var _ = Describe("Main", func() {
	var n Network
	var ch chan *stan.Msg
	// var chStr chan string
	var natsConn stan.Conn
	var natsSub stan.Subscription
	var err error

	var chPolling chan string
	//type chFunc func(ch chan string) error

	//var pollingFunc func(timeout int, model provide.IModel) error
	//var pollingFunc func(timeout int, f chFunc) error
	// pollingFunc := func(timeout time.Duration, chFunc func(ch chan string) error, pollingInterval ...time.Duration) error {
	// 	startedAt := time.Now().UnixNano()
	// 	interval := (time.Millisecond * 100)
	// 	if len(pollingInterval) > 0 {
	// 		interval = pollingInterval[0]
	// 	}
	// 	ticker := time.NewTicker(interval)
	// 	timer := time.NewTimer(timeout)

	// 	elapsedMillis := (time.Now().UnixNano() - startedAt) / 1000000
	// 	Log.Debugf("ticker: %d", elapsedMillis)
	// 	go func() error {
	// 		for {
	// 			select {
	// 			case <-ticker.C:
	// 				elapsedMillis := (time.Now().UnixNano() - startedAt) / 1000000
	// 				Log.Debugf("ticker: %d", elapsedMillis)
	// 				if elapsedMillis >= int64(timeout) {
	// 					ticker.Stop()
	// 				}

	// 				return chFunc(chPolling)
	// 			case <-timer.C:
	// 				return nil
	// 			default:
	// 				// no-op
	// 			}
	// 		}
	// 	}()
	// 	return nil
	// }

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

						Context("with chainspec and chainspec abi", func() {
							BeforeEach(func() {
								// TODO move parsing to functions
								ethChainspecFileurl := "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn/spec.json"
								ethChainspecAbiFileurl := "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json"
								response, err := http.Get(ethChainspecFileurl)
								//chainspec_text := ""
								// chainspec_abi_text := ""
								chainspecJSON := map[string]interface{}{}
								chainspecABIJSON := map[string]interface{}{}

								if err != nil {
									fmt.Printf("%s\n", err)
								} else {
									defer response.Body.Close()
									contents, err := ioutil.ReadAll(response.Body)
									if err != nil {
										fmt.Printf("%s\n", err)
									}
									// fmt.Printf("%s\n", string(contents))
									//chainspec_text = string(contents)
									errJSON := json.Unmarshal(contents, &chainspecJSON)
									Log.Debugf("error parsing chainspec: %v", errJSON)

								}

								responseAbi, err := http.Get(ethChainspecAbiFileurl)

								if err != nil {
									fmt.Printf("%s\n", err)
								} else {
									defer responseAbi.Body.Close()
									contents, err := ioutil.ReadAll(responseAbi.Body)
									if err != nil {
										fmt.Printf("%s\n", err)
									}
									// fmt.Printf("%s\n", string(contents))
									// chainspec_abi_text = string(contents)
									errJSON := json.Unmarshal(contents, &chainspecABIJSON)
									Log.Debugf("error parsing chainspec: %v", errJSON)
								}

								n.Config = marshalConfig(map[string]interface{}{
									"block_explorer_url":  "https://unicorn-explorer.provide.network",
									"chain":               "unicorn-v0",
									"chainspec":           chainspecJSON,
									"chainspec_abi":       chainspecABIJSON,
									"cloneable_cfg":       map[string]interface{}{},
									"engine_id":           "authorityRound", // required
									"is_ethereum_network": true,
									"is_load_balanced":    false,
									"json_rpc_url":        nil,
									"native_currency":     "PRVD", // required
									"network_id":          22,
									"protocol_id":         "poa", // required
									"websocket_url":       nil})

								chPolling = make(chan string, 1)

								cf := func(ch chan string) error {
									db := dbconf.DatabaseConnection()
									//db.Model( &(reflect.TypeOf(m)){} ).Count(&count)

									objects := []Contract{}
									db.Find(&objects)

									for _, object := range objects {
										ch <- object.ID.String()
									}

									return nil
								}

								pollingToStrChFunc(chPolling, cf, nil) // last param to receive default message "timeout"
							})

							It("shoud be valid", func() {
								Expect(n.Validate()).To(BeTrue())
							})

							It("should create successfully and send message to NATS", func() {
								Expect(n.Create()).To(BeTrue())

								Eventually(chPolling).Should(Receive(Equal("timeout"))) // ending of Contracts processing and sending their IDs to pipe

								// getting all contracts IDs from channel
								// s := make([]string, 0)
								// for i := range chPolling {
								// 	s = append(s, i)
								// }

								objects := []Contract{}
								db := dbconf.DatabaseConnection()
								db.Find(&objects)

								Expect(objects).To(HaveLen(1))
								//Log.Debugf("%v", objects[0])
								Expect(objects[0]).To(MatchFields(IgnoreExtras, Fields{
									"Model": MatchFields(IgnoreExtras, Fields{
										"ID":        Not(BeNil()),
										"CreatedAt": Not(BeNil()),
										"Errors":    BeEmpty(),
									}),
									"NetworkID":     Equal(n.ID),
									"ApplicationID": Equal(n.ApplicationID),
									"ContractID":    BeNil(),
									"TransactionID": BeNil(),
									"Name":          PointTo(Equal("Network Contract 0x0000000000000000000000000000000000000017")),
									"Address":       PointTo(Equal("0x0000000000000000000000000000000000000017")),
									// "Params":        PointTo(Equal("")), // TODO add params body
									"AccessedAt": BeNil(),
								}))

								// count := 0
								// db := dbconf.DatabaseConnection()
								// db.Model(&Contract{}).Count(&count)
								// Expect(count).To(Equal(1))
							})
						})

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
