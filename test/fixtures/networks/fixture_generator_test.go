package networkfixtures_test

import (
	"encoding/json"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/provideapp/goldmine/common"
	networkfixtures "github.com/provideapp/goldmine/test/fixtures/networks"
)

func defaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"block_explorer_url": "https://unicorn-explorer.provide.network", // required
		"chain":              "unicorn-v0",                               // required
		"chainspec_abi_url":  "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
		"chainspec_url":      "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json", // required If ethereum network
		"cloneable_cfg": map[string]interface{}{
			"security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security
		"engine_id":           "aura", // required
		"is_ethereum_network": true,   // required for ETH
		"is_load_balanced":    true,   // implies network load balancer count > 0
		"json_rpc_url":        nil,
		"native_currency":     "PRVD", // required
		"network_id":          22,     // required
		"protocol_id":         "poa",  // required
		"websocket_url":       nil}
}
func TestNetworkFixtures(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network Fixtures Suite")
}

func ptrToBool1(b bool) *bool {
	return &b
}

func ptrTo(s string) *string {
	return &s
}

func ptrToInt(i int) *int {
	return &i
}

var _ = Describe("Network Fixture Generator", func() {

	var nfg *networkfixtures.NetworkFixtureGenerator

	// Context("Generate()", func() {
	// 	Context("empty values", func() {
	// 		BeforeEach(func() {
	// 			nfg = &networkfixtures.NetworkFixtureGenerator{}
	// 			// nfg = networkfixtures.NewNetworkFixtureGenerator([]*networkfixtures.NetworkFixtureFieldValues{}, []*string{})
	// 		})

	// 		It("should return empty array", func() {
	// 			// Expect(nfg.Generate()).To(Panic()) -- doesn't work
	// 		})
	// 	})

	// 	FContext("dumb", func() {
	// 		It("should pass", func() {
	// 			Expect(0).To(Equal(0))
	// 		})
	// 	})

	Context("empty arguments", func() {
		BeforeEach(func() {
			nfg = networkfixtures.NewNetworkFixtureGenerator(
				[]*networkfixtures.NetworkFixtureFieldValues{},
			)
		})

		It("should return empty response", func() {
			Expect(nfg.Generate()).To(HaveLen(0))
		})
	})

	Context("1 field", func() {
		Describe("which is Name/Prefix", func() {
			Describe("with one variant", func() {
				BeforeEach(func() {
					nfg = networkfixtures.NewNetworkFixtureGenerator(
						[]*networkfixtures.NetworkFixtureFieldValues{
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Name/Prefix"),
								[]interface{}{
									common.StringOrNil("eth")},
							),
						},
					)
				})

				It("should return 1 fixture", func() {
					Expect(nfg.Generate()).To(HaveLen(1))
				})
				It("should return NetworkField with correct values", func() {
					res := nfg.Generate()
					config := defaultConfig()
					cfgJSON, _ := json.Marshal(config)
					_cfgJSON := json.RawMessage(cfgJSON)

					Expect(*res[0]).To(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Config w cloneable_cfg w chainspec_url w block_explorer_url w chain w engine_id eth:t lb:t PRVD currency 22 network_id poa protocol_id ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       gstruct.PointTo(Equal(_cfgJSON)),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))
				})
			})
			Describe("with two variants", func() {
				BeforeEach(func() {
					nfg = networkfixtures.NewNetworkFixtureGenerator(
						[]*networkfixtures.NetworkFixtureFieldValues{
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Name/Prefix"),
								[]interface{}{
									ptrTo(""),
									ptrTo("eth")},
							),
						},
					)
				})

				It("should return 2 fixture", func() {
					Expect(nfg.Generate()).To(HaveLen(2))
				})
				It("should return NetworkField with correct values", func() {
					res := nfg.Generate()
					config := defaultConfig()
					cfgJSON, _ := json.Marshal(config)
					_cfgJSON := json.RawMessage(cfgJSON)

					Expect(*res[0]).To(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal(" NonProduction NonCloneable Disabled Config w cloneable_cfg w chainspec_url w block_explorer_url w chain w engine_id eth:t lb:t PRVD currency 22 network_id poa protocol_id ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       gstruct.PointTo(Equal(_cfgJSON)),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))
					Expect(*res[1]).To(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Config w cloneable_cfg w chainspec_url w block_explorer_url w chain w engine_id eth:t lb:t PRVD currency 22 network_id poa protocol_id ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       gstruct.PointTo(Equal(_cfgJSON)),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))
				})
			})
		})
	})

	Context("2 field", func() {
		Describe("which is Name/Prefix and Config", func() {
			Describe("with one variant", func() {
				BeforeEach(func() {
					nfg = networkfixtures.NewNetworkFixtureGenerator(
						[]*networkfixtures.NetworkFixtureFieldValues{
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Name/Prefix"),
								[]interface{}{
									common.StringOrNil("eth")},
							),
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Config"),
								[]interface{}{
									nil},
							),
						},
					)
				})

				It("should return 1 fixture", func() {
					Expect(nfg.Generate()).To(HaveLen(1))
				})
				It("should return NetworkField with correct values", func() {
					res := nfg.Generate()
					Expect(*res[0]).To(gstruct.MatchAllFields(gstruct.Fields{
						//"Name":         gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Config w cloneable_cfg w chainspec_url w block_explorer_url w chain w engine_id eth:t lb:t PRVD currency 22 network_id poa protocol_id ")),

						"Name": gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Nil Config ")),

						//"Name":         BeNil(),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       BeNil(),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))
				})
			})
			Describe("with two variants", func() {
				BeforeEach(func() {
					nfg = networkfixtures.NewNetworkFixtureGenerator(
						[]*networkfixtures.NetworkFixtureFieldValues{
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Name/Prefix"),
								[]interface{}{
									ptrTo(""),
									ptrTo("eth")},
							),
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Config"),
								[]interface{}{
									nil,
									map[string]interface{}{},
								},
							),
						},
					)
				})

				It("should return 4 fixture", func() {
					Expect(nfg.Generate()).To(HaveLen(4))
				})
				It("should return NetworkField with correct values", func() {
					res := nfg.Generate()
					config := map[string]interface{}{}
					cfgJSON, _ := json.Marshal(config)
					_cfgJSON := json.RawMessage(cfgJSON)

					Expect(res).To(ContainElement(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal(" NonProduction NonCloneable Disabled Nil Config ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       BeNil(),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))))
					Expect(res).To(ContainElement(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal(" NonProduction NonCloneable Disabled Empty Config ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       gstruct.PointTo(Equal(_cfgJSON)),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))))

					Expect(res).To(ContainElement(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Empty Config ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       gstruct.PointTo(Equal(_cfgJSON)),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))))

					Expect(res).To(ContainElement(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Nil Config ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       BeNil(),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))))
				})
			})
		})
	})

	Context("Config", func() {
		ptrFalse := ptrToBool1(false)
		ptrTrue := ptrToBool1(true)
		Context("empty values ", func() {

			BeforeEach(func() {
				//ptrTrue := ptrToBool1(true)

				nfg = networkfixtures.NewNetworkFixtureGenerator(
					[]*networkfixtures.NetworkFixtureFieldValues{
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Name/Prefix"),
							[]interface{}{
								ptrTo("eth")},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config"),
							[]interface{}{
								//nil,
								map[string]interface{}{},
							},
						),

						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/Skip"),
							[]interface{}{
								ptrFalse,
								// ptrTrue
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/cloneable_cfg"),
							[]interface{}{
								//nil,
								map[string]interface{}{},
								//map[string]interface{}{
								//	"_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security,
							},
						),

						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/chainspec_url"),
							[]interface{}{
								// nil,
								ptrTo(""),
								//ptrTo("https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json"),
								// ptrTo("get https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/block_explorer_url"),
							[]interface{}{
								//nil,
								ptrTo(""),
								// ptrTo("https://unicorn-explorer.provide.network"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/chain"),
							[]interface{}{
								// nil,
								ptrTo(""),
								// ptrTo("unicorn-v0"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/engine_id"),
							[]interface{}{
								// nil,
								ptrTo(""),
								// ptrTo("aura"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_ethereum_network"),
							[]interface{}{
								//nil,
								ptrFalse,
								//ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_bcoin_network"),
							[]interface{}{
								// nil,
								ptrFalse,
								// ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_handshake_network"),
							[]interface{}{
								// nil,
								ptrFalse,
								// ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_lcoin_network"),
							[]interface{}{
								// nil,
								ptrFalse,
								// ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_quorum_network"),
							[]interface{}{
								// nil,
								ptrFalse,
								// ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_load_balanced"),
							[]interface{}{
								// nil,
								ptrFalse,
								// ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/native_currency"),
							[]interface{}{
								// nil,
								ptrTo(""),
								// ptrTo("PRVD"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/network_id"),
							[]interface{}{
								// nil,
								ptrTo(""),
								// ptrToInt(0),
								// ptrToInt(22),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/protocol_id"),
							[]interface{}{
								// nil,
								ptrTo(""),
								// ptrTo("poa"),
							},
						),
					},
				)
			})

			It("should match empty values config", func() {
				res := nfg.Generate()
				config := map[string]interface{}{
					"block_explorer_url":  "", // required
					"chain":               "", // required
					"chainspec_abi_url":   "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
					"chainspec_url":       "",                       // required If ethereum network
					"cloneable_cfg":       map[string]interface{}{}, // If cloneable CFG then security
					"engine_id":           "",                       // required
					"is_ethereum_network": false,                    // required for ETH
					"is_load_balanced":    false,                    // implies network load balancer count > 0
					"json_rpc_url":        nil,
					"native_currency":     "", // required
					"network_id":          "", // required
					"protocol_id":         "", // required
					"websocket_url":       nil,
				}
				cfgJSON, _ := json.Marshal(config)
				_cfgJSON := json.RawMessage(cfgJSON)

				Expect(*res[0]).To(gstruct.MatchAllFields(gstruct.Fields{
					"Name": gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Config empty cloneable_cfg empty chainspec_url w block_explorer_url w chain w engine_id eth:f lb:f empty currency empty network_id empty protocol_id ")),

					//"Name":         BeNil(),
					"IsProduction": BeNil(),
					"Cloneable":    BeNil(),
					"Enabled":      BeNil(),
					"Config":       gstruct.PointTo(Equal(_cfgJSON)),
					"Description":  BeNil(),
					"ChainID":      BeNil(),
					"Model": gstruct.MatchAllFields(gstruct.Fields{
						"Errors":    BeEmpty(),
						"ID":        Equal(uuid.Nil),
						"CreatedAt": Equal(time.Time{}),
					}),
				}))
			})
		})
		Context("full values ", func() {
			BeforeEach(func() {
				//ptrTrue := ptrToBool1(true)
				ptrFalse := ptrToBool1(false)
				nfg = networkfixtures.NewNetworkFixtureGenerator(
					[]*networkfixtures.NetworkFixtureFieldValues{
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Name/Prefix"),
							[]interface{}{
								ptrTo("eth")},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config"),
							[]interface{}{
								//nil,
								map[string]interface{}{},
							},
						),

						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/Skip"),
							[]interface{}{
								ptrFalse,
								// ptrTrue
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/cloneable_cfg"),
							[]interface{}{
								//nil,
								//map[string]interface{}{},
								map[string]interface{}{
									"_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security,
							},
						),

						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/chainspec_url"),
							[]interface{}{
								// nil,
								//ptrTo(""),
								ptrTo("https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json"),
								// ptrTo("get https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/block_explorer_url"),
							[]interface{}{
								//nil,
								//ptrTo(""),
								ptrTo("https://unicorn-explorer.provide.network"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/chain"),
							[]interface{}{
								// nil,
								//ptrTo(""),
								ptrTo("unicorn-v0"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/engine_id"),
							[]interface{}{
								// nil,
								//ptrTo(""),
								ptrTo("aura"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_ethereum_network"),
							[]interface{}{
								//nil,
								//ptrFalse,
								ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_bcoin_network"),
							[]interface{}{
								// nil,
								//ptrFalse,
								ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_handshake_network"),
							[]interface{}{
								// nil,
								//ptrFalse,
								ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_lcoin_network"),
							[]interface{}{
								// nil,
								//ptrFalse,
								ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_quorum_network"),
							[]interface{}{
								// nil,
								//ptrFalse,
								ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/is_load_balanced"),
							[]interface{}{
								// nil,
								//ptrFalse,
								ptrTrue,
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/native_currency"),
							[]interface{}{
								// nil,
								//ptrTo(""),
								ptrTo("PRVD"),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/network_id"),
							[]interface{}{
								// nil,
								//ptrTo(""),
								// ptrToInt(0),
								ptrToInt(22),
							},
						),
						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/protocol_id"),
							[]interface{}{
								// nil,
								//ptrTo(""),
								ptrTo("poa"),
							},
						),
					},
				)
			})

			It("should match full config", func() {
				res := nfg.Generate()
				config := map[string]interface{}{
					"block_explorer_url": "https://unicorn-explorer.provide.network", // required
					"chain":              "unicorn-v0",                               // required
					"chainspec_abi_url":  "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
					"chainspec_url":      "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json", // required If ethereum network
					"cloneable_cfg": map[string]interface{}{
						"_security": map[string]interface{}{
							"egress": "*",
							"ingress": map[string]interface{}{
								"0.0.0.0/0": map[string]interface{}{
									"tcp": [5]int{5001, 8050, 8051, 8080, 30300},
									"udp": [1]int{30300},
								},
							},
						},
					}, // If cloneable CFG then security
					// "cloneable_cfg":  "{"_security":{"egress":"*","ingress":{"0.0.0.0/0":{"tcp":[5001,8050,8051,8080,30300],"udp":[30300]}}}}",
					"engine_id":           "aura", // required
					"is_ethereum_network": true,   // required for ETH
					"is_load_balanced":    true,   // implies network load balancer count > 0
					"json_rpc_url":        nil,
					"native_currency":     "PRVD", // required
					"network_id":          22,     // required
					"protocol_id":         "poa",  // required
					"websocket_url":       nil,
				}
				cfgJSON, _ := json.Marshal(config)
				_cfgJSON := json.RawMessage(cfgJSON)

				Expect(*res[0]).To(gstruct.MatchAllFields(gstruct.Fields{
					"Name": gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Config w cloneable_cfg w chainspec_url w block_explorer_url w chain w engine_id eth:t lb:t PRVD currency 22 network_id poa protocol_id ")),

					//"Name":         BeNil(),
					"IsProduction": BeNil(),
					"Cloneable":    BeNil(),
					"Enabled":      BeNil(),
					"Config":       gstruct.PointTo(Equal(_cfgJSON)),
					"Description":  BeNil(),
					"ChainID":      BeNil(),
					"Model": gstruct.MatchAllFields(gstruct.Fields{
						"Errors":    BeEmpty(),
						"ID":        Equal(uuid.Nil),
						"CreatedAt": Equal(time.Time{}),
					}),
				}))
			})
		})

		Describe("nil Config", func() {
			Describe("with one variant", func() {
				BeforeEach(func() {
					nfg = networkfixtures.NewNetworkFixtureGenerator(
						[]*networkfixtures.NetworkFixtureFieldValues{
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Name/Prefix"),
								[]interface{}{
									common.StringOrNil("eth")},
							),
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Config"),
								[]interface{}{
									nil},
							),
						},
					)
				})

				It("should return 1 fixture", func() {
					Expect(nfg.Generate()).To(HaveLen(1))
				})
				It("should return NetworkField with correct values", func() {
					res := nfg.Generate()
					Expect(*res[0]).To(gstruct.MatchAllFields(gstruct.Fields{
						//"Name":         gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Config w cloneable_cfg w chainspec_url w block_explorer_url w chain w engine_id eth:t lb:t PRVD currency 22 network_id poa protocol_id ")),

						"Name": gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Nil Config ")),

						//"Name":         BeNil(),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       BeNil(),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))
				})
			})
			Describe("with two variants", func() {
				BeforeEach(func() {
					nfg = networkfixtures.NewNetworkFixtureGenerator(
						[]*networkfixtures.NetworkFixtureFieldValues{
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Name/Prefix"),
								[]interface{}{
									ptrTo(""),
									ptrTo("eth")},
							),
							networkfixtures.NewNetworkFixtureFieldValues(
								common.StringOrNil("Config"),
								[]interface{}{
									nil,
									map[string]interface{}{},
								},
							),
						},
					)
				})

				It("should return 4 fixture", func() {
					Expect(nfg.Generate()).To(HaveLen(4))
				})
				It("should return NetworkField with correct values", func() {
					res := nfg.Generate()
					config := map[string]interface{}{}
					cfgJSON, _ := json.Marshal(config)
					_cfgJSON := json.RawMessage(cfgJSON)

					Expect(res).To(ContainElement(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal(" NonProduction NonCloneable Disabled Nil Config ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       BeNil(),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))))
					Expect(res).To(ContainElement(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal(" NonProduction NonCloneable Disabled Empty Config ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       gstruct.PointTo(Equal(_cfgJSON)),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))))

					Expect(res).To(ContainElement(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Empty Config ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       gstruct.PointTo(Equal(_cfgJSON)),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))))

					Expect(res).To(ContainElement(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
						"Name":         gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Nil Config ")),
						"IsProduction": BeNil(),
						"Cloneable":    BeNil(),
						"Enabled":      BeNil(),
						"Config":       BeNil(),
						"Description":  BeNil(),
						"ChainID":      BeNil(),
						"Model": gstruct.MatchAllFields(gstruct.Fields{
							"Errors":    BeEmpty(),
							"ID":        Equal(uuid.Nil),
							"CreatedAt": Equal(time.Time{}),
						}),
					}))))
				})
			})
		})
	})

	Context("minimal set of fields", func() {
		BeforeEach(func() {
			nfg = networkfixtures.NewNetworkFixtureGenerator(
				[]*networkfixtures.NetworkFixtureFieldValues{
					networkfixtures.NewNetworkFixtureFieldValues(
						common.StringOrNil("Name/Prefix"),
						[]interface{}{
							common.StringOrNil("eth")},
					),
					networkfixtures.NewNetworkFixtureFieldValues(
						common.StringOrNil("IsProduction"),
						[]interface{}{
							common.BoolOrNil(false)},
					),
					networkfixtures.NewNetworkFixtureFieldValues(
						common.StringOrNil("Cloneable"),
						[]interface{}{
							common.BoolOrNil(false)},
					),
					networkfixtures.NewNetworkFixtureFieldValues(
						common.StringOrNil("Enabled"),
						[]interface{}{
							common.BoolOrNil(false)},
					),
					networkfixtures.NewNetworkFixtureFieldValues(
						common.StringOrNil("Config"),
						[]interface{}{
							nil},
					),
				},
			)
		})

		It("should return 1 NetworkFields object", func() {
			Expect(nfg.Generate()).To(HaveLen(1))
		})

		It("should return NetworkField with correct values", func() {
			res := nfg.Generate()

			Expect(*res[0]).To(gstruct.MatchAllFields(gstruct.Fields{
				"Name":         gstruct.PointTo(Equal("eth NonProduction NonCloneable Disabled Nil Config ")),
				"IsProduction": gstruct.PointTo(BeFalse()),
				"Cloneable":    gstruct.PointTo(BeFalse()),
				"Enabled":      gstruct.PointTo(BeFalse()),
				"Config":       BeNil(),
				"Description":  BeNil(),
				"ChainID":      BeNil(),
				"Model": gstruct.MatchAllFields(gstruct.Fields{
					"Errors":    BeEmpty(),
					"ID":        Equal(uuid.Nil),
					"CreatedAt": Equal(time.Time{}),
				}),
			}))
		})
	})

	// })

})
