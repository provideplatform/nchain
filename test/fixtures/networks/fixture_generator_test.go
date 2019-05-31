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

func ptrToBool1(b bool) *bool {
	return &b
}

func ptrTo(s string) *string {
	return &s
}

func defaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"block_explorer_url": "https://unicorn-explorer.provide.network", // required
		"chain":              "unicorn-v0",                               // required
		"chainspec_abi_url":  "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
		"chainspec_url":      "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json", // required If ethereum network
		"cloneable_cfg": map[string]interface{}{
			"security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security
		"engine_id":           "authorityRound", // required
		"is_ethereum_network": true,             // required for ETH
		"is_load_balanced":    true,             // implies network load balancer count > 0
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

var _ = Describe("Network Fixture Generator", func() {
	var nfg *networkfixtures.NetworkFixtureGenerator

	ptrTrue := ptrToBool1(true)
	ptrFalse := ptrToBool1(false)

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
			Describe("which is one variant", func() {
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
			Describe("which is two variants", func() {
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
			Describe("which is one variant", func() {
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
			Describe("which is two variants", func() {
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
		Context("nil values ", func() {
			BeforeEach(func() {
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
								nil,
								map[string]interface{}{}},
						),

						networkfixtures.NewNetworkFixtureFieldValues(
							common.StringOrNil("Config/Skip"),
							[]interface{}{
								ptrFalse,
								ptrTrue},
						),
					},
				)
			})
		})
		Context("empty values ", func() {
		})
		Context("full values ", func() {
		})

		Describe("nil  empty full Config", func() {
			Describe("which is one variant", func() {
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
			Describe("which is two variants", func() {
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
