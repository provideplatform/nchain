// +build unit

package network_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/contract"
	"github.com/provideapp/goldmine/network"
	"github.com/provideapp/goldmine/test"
	networkfixtures "github.com/provideapp/goldmine/test/fixtures/networks"
	"github.com/provideapp/goldmine/test/matchers"

	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	stan "github.com/nats-io/go-nats-streaming"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	//. "github.com/onsi/gomega/gstruct"
)

func ptrTo(s string) *string {
	return &s
}

func ptrToBool(b bool) *bool {
	return &b
}

func testNetworks() (nf []*networkFactory, nc []*networkfixtures.NetworkFixture) {
	// ns = make([]map[string]interface{}, 0)
	// for _, nf := range networkfixtures.Networks() {
	// 	n, s := networkFactory(nf.Fixture.(*networkfixtures.NetworkFixture))
	// 	fmt.Printf("%v", n)
	// 	// common.Log.Debugf("%s", n)

	// 	ns = append(ns, map[string]interface{}{
	// 		"matchers": nf.Matcher,
	// 		"network":  n,
	// 		"name":     s,
	// 	})
	// }
	// return

	networkFixtureGenerator := networkfixtures.NewNetworkFixtureGenerator(nil)
	dispatcher := networkfixtures.NewNetworkFixtureDispatcher(networkFixtureGenerator)

	networks := dispatcher.Networks()
	count := len(networks)
	nf = make([]*networkFactory, count)
	nc = dispatcher.NotCovered()

	for i, n := range networks {
		fixture := n.Fixture.(*networkfixtures.NetworkFixture)
		nf[i] = &networkFactory{
			fixture:  fixture,
			Name:     fixture.Name,
			Matchers: n.Matcher,
		}
	}
	return
}

func clearNetworks(db *gorm.DB) {
	uid, _ := uuid.FromString("36a5f8e0-bfc1-49f8-a7ba-86457bb52912")
	db.Delete(network.Network{}, "id != ?", uid)
}

type networkFactory struct {
	fixture  *networkfixtures.NetworkFixture
	Name     *string
	Matchers *matchers.MatcherCollection
}

func (factory *networkFactory) Network() (n *network.Network) {
	nf := factory.fixture.Fields
	n = &network.Network{
		// ApplicationID: nf.ApplicationID,
		// UserID:        nf.UserID,
		Name:         nf.Name,
		Description:  nf.Description,
		IsProduction: nf.IsProduction,
		Cloneable:    nf.Cloneable,
		Enabled:      nf.Enabled,
		ChainID:      nf.ChainID,
		// SidechainID:   nf.SidechainID,
		// NetworkID:     nf.NetworkID,
		Config: nf.Config,
		// Stats:         nf.Stats,
	}
	return
}
func TestNetworks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network Suite")
}

var _ = Describe("Network", func() {

	var n *network.Network
	var mc *matchers.MatcherCollection
	// var ch chan *stan.Msg
	// var chStr chan string
	// var natsConn stan.Conn
	var natsSub stan.Subscription
	// var err error

	var chPolling chan string

	var networks, rest = testNetworks()

	Describe("Network", func() {
		Context("production", func() {})

		Context("network fixtures", func() {
			It("should cover all generator cases", func() {
				// fixtures := networkFixtureGenerator.All()
				// Expect(len(fixtures) - len(networks)).To(Equal(0))
				// Expect(fixtures).To(HaveLen(8))
				names := make([]*string, len(rest))
				for i, f := range rest {
					names[i] = f.Name
				}
				Expect(names).To(HaveLen(0))
			})
		})

		// TODO:
		//   1. add mocks to check NATS and other calls (we can't just check all NATS channels to see nothing is written)
		//   2. add behaviors for private methods
		//   3. add config keys to generator
		Context("Dynamic", func() {

			for i := 0; i < len(networks); i++ {

				nn := networks[i] // current network being tested
				name := *nn.Name  // current network name

				Context(name, func() { // context for current network

					BeforeEach(func() {
						var count int
						db := dbconf.DatabaseConnection()
						db.Model(&contract.Contract{}).Count(&count)
						// fmt.Printf("contract count: %v\n", count)
						// fmt.Printf("purging contracts\n")

						db.Delete(contract.Contract{})
						clearNetworks(db)
						db.Model(&contract.Contract{}).Count(&count)
						// fmt.Printf("contract count: %v\n", count)

						n = nn.Network() // creating new pointer with network data for each test
						mc = nn.Matchers // set of matchers for current network
					})

					AfterEach(func() {
						db := dbconf.DatabaseConnection()

						db.Delete(contract.Contract{})

						if natsSub != nil {
							natsSub.Unsubscribe()
						}
					})

					// Context("NATS", func() {
					// 	BeforeEach(func() {

					// 		matcherName := "Create with NATS"
					// 		var chName string
					// 		if opts, ok := mc.MatcherOptionsFor(matcherName); ok {
					// 			chName = *opts.NATSChannels[0]
					// 		}

					// 		natsConn = common.GetDefaultNatsStreamingConnection()
					// 		ch = make(chan *stan.Msg, 1)
					// 		natsSub, err = natsConn.QueueSubscribe(chName, chName, func(msg *stan.Msg) {
					// 			ch <- msg
					// 		})
					// 		if err != nil {
					// 			common.Log.Debugf("conn failure")
					// 		}

					// 		test.NatsGuaranteeDelivery(chName)
					// 	})
					// 	It("should catch NATS message", func() {
					// 		chPolling = make(chan string, 1)
					// 		cf := func(ch chan string) error {
					// 			return nil
					// 		}
					// 		test.PollingToStrChFunc(chPolling, cf, nil)

					// 		matcherName := "Create with NATS"
					// 		Expect(n.Create()).To(mc.MatchBehaviorFor(matcherName, chPolling))
					// 	})
					// })

					Context("channeling", func() {
						It("should be created", func() {
							chPolling = make(chan string, 1)

							matcherName := "Create"
							var funcAfter func() []interface{}

							// if options, ok := mc.MatcherOptionsFor(matcherName); ok {
							// 	if options.ChannelPolling {
							cf := func(ch chan string) error {
								db := dbconf.DatabaseConnection()
								//db.Model( &(reflect.TypeOf(m)){} ).Count(&count)

								objects := []contract.Contract{}
								db.Find(&objects)

								for i, object := range objects {
									fmt.Printf("%vth object ID: %v\n", i, object.ID.String())
									ch <- object.ID.String()
								}

								return nil
							}

							test.PollingToStrChFunc(chPolling, cf, nil) // last param nil to receive default message "timeout"
							// 	}
							// }

							funcAfter = func() []interface{} {
								objects := []contract.Contract{}
								ptrs := []interface{}{}
								db := dbconf.DatabaseConnection()
								db.Find(&objects)
								for _, o := range objects {
									ptrs = append(ptrs, &o)
								}
								return ptrs
							}

							Expect(n).To(mc.MatchBehaviorFor(matcherName, n, chPolling, funcAfter))

							// created := n.Create()
							// Expect(created).To(BeTrue())
							// Expect(n.Errors).To(BeEmpty())
						})
					})

					It("should be valid", func() {
						Expect(n).To(mc.MatchBehaviorFor("Validate", n))
					})
					It("should parse config", func() {
						Expect(n.ParseConfig()).To(mc.MatchBehaviorFor("ParseConfig"))
					})
					It("should return network type correctly", func() {
						Expect(n.IsEthereumNetwork()).To(mc.MatchBehaviorFor("Network type", "eth"))
						Expect(n.IsBcoinNetwork()).To(mc.MatchBehaviorFor("Network type", "btc"))
						Expect(n.IsHandshakeNetwork()).To(mc.MatchBehaviorFor("Network type", "handshake"))
						Expect(n.IsLcoinNetwork()).To(mc.MatchBehaviorFor("Network type", "ltc"))
						Expect(n.IsQuorumNetwork()).To(mc.MatchBehaviorFor("Network type", "quorum"))
					})
					It("should not create second record", func() {
						n.Create()
						Expect(n.Create()).To(mc.MatchBehaviorFor("Double Create"))
					})
					It("should return RPC URL", func() {
						Expect(n.RpcURL()).To(mc.MatchBehaviorFor("RpcURL"))
					})
					It("should reload instance", func() {
						// Expect(n.Reload()).To(mc.MatchBehaviorFor("Reload")) // FIXME
					})
					It("should update instance", func() {
						// Expect(n.Update()).To(mc.MatchBehaviorFor("Update")) // FIXME
					})
					It("should set config", func() {
						// private
					})
					It("should set chain id", func() {
						// private
					})
					It("should get security configuration", func() {
						// private
						// Expect(n.getSecurityConfiguration()).To(mc.MatchBehaviorFor("securityConfiguration"))
					})
					It("should resolve and balance JSON RPC and Websocket", func() {
						// Expect(n.resolveAndBalanceJSONRPCAndWebsocketURLs()).To(mc.MatchBehaviorFor("resolveAndBalanceJSONRPCAndWebsocketURLs"))
					})
					It("should return load balancers", func() {
						// Expect(n.LoadBalancers()).To(mc.MatchBehaviorFor("LoadBalancers"))
					})
					It("should invoke JSON RPC", func() {
						// Expect(n.InvokeJSONRPC()).To(mc.MatchBehaviorFor("InvokeJSONRPC"))
					})
					It("should return network status", func() {
						// Expect(n.Status()).To(mc.MatchBehaviorFor("Status"))
					})
					It("should return NodeCount", func() {
						Expect(n.NodeCount()).To(mc.MatchBehaviorFor("NodeCount"))
					})
					It("should return AvailablePeerCount", func() {
						Expect(n.AvailablePeerCount()).To(mc.MatchBehaviorFor("AvailablePeerCount"))
					})
					It("should return bootnodes txt", func() {
						// Expect(n.BootnodesTxt()).To(mc.MatchBehaviorFor("BootnodesTxt"))
					})
					It("should return bootnodes count", func() {
						// Expect(n.BootnodesCount()).To(mc.MatchBehaviorFor("BootnodesCount"))
					})
					It("should return bootnodes", func() {
						// Expect(n.Bootnodes()).To(mc.MatchBehaviorFor("Bootnodes"))
					})
					It("should return nodes", func() {
						// Expect(n.Nodes()).To(mc.MatchBehaviorFor("Nodes"))
					})

				})
			}
		})
		/* alex
		Context("with LB", func() {
			var nlb *network.Network
			var lb, lb2 *network.LoadBalancer
			BeforeEach(func() {

				nlb = &network.Network{
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
					Config: common.MarshalConfig(map[string]interface{}{
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
						"websocket_url":       nil}),
					Stats: nil}

				// ch = make(chan *stan.Msg, 1)
				nlb.Create()

				lbConfig := map[string]interface{}{
					"json_rpc_url": "url",
				}
				lb = &network.LoadBalancer{
					NetworkID: nlb.ID,
					Name:      common.StringOrNil("LB"),
					Type:      common.StringOrNil("rpc"),
					Config:    common.MarshalConfig(lbConfig),
				}
				r := lb.Create()
				fmt.Printf("load balancer created: %t\n", r)

				lb2 = &network.LoadBalancer{
					NetworkID: nlb.ID,
					Name:      common.StringOrNil("LB2"),
					Type:      common.StringOrNil("websocket"),
					Region:    common.StringOrNil("region"),
					Config:    common.MarshalConfig(lbConfig),
				}
				lb2.Create()

			})

			AfterEach(func() {
				db := dbconf.DatabaseConnection()
				clearNetworks(db)
				db.Delete(network.LoadBalancer{})
			})

			Context("LoadBalancers()", func() {
				It("should return all load balancer", func() {
					db := dbconf.DatabaseConnection()

					lbResult, lbErr := nlb.LoadBalancers(db, nil, nil)

					Expect(lbResult[1]).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"NetworkID": Equal(nlb.ID),
						"Name":      gstruct.PointTo(Equal("LB")),
						"Type":      gstruct.PointTo(Equal("rpc")),
					})))
					Expect(lbResult[0]).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"NetworkID": Equal(nlb.ID),
						"Name":      gstruct.PointTo(Equal("LB2")),
						"Region":    gstruct.PointTo(Equal("region")),
						"Type":      gstruct.PointTo(Equal("websocket")),
					})))
					Expect(lbResult).To(HaveLen(2))
					Expect(lbErr).NotTo(HaveOccurred())
				})

				It("should return all load balancer", func() {
					db := dbconf.DatabaseConnection()

					lbResult, lbErr := nlb.LoadBalancers(db, nil, lb.Type)

					Expect(lbResult[0]).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"NetworkID": Equal(nlb.ID),
						"Name":      gstruct.PointTo(Equal("LB")),
						"Type":      gstruct.PointTo(Equal("rpc")),
					})))
					Expect(lbResult).To(HaveLen(1))
					Expect(lbErr).NotTo(HaveOccurred())
				})

				It("should return all load balancer", func() {
					db := dbconf.DatabaseConnection()

					lbResult, lbErr := nlb.LoadBalancers(db, nil, lb2.Type)

					Expect(lbResult[0]).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"NetworkID": Equal(nlb.ID),
						"Name":      gstruct.PointTo(Equal("LB2")),
						"Region":    gstruct.PointTo(Equal("region")),
						"Type":      gstruct.PointTo(Equal("websocket")),
					})))
					Expect(lbResult).To(HaveLen(1))
					Expect(lbErr).NotTo(HaveOccurred())
				})

				It("should return load balancer with region", func() {
					db := dbconf.DatabaseConnection()

					lbResult, lbErr := nlb.LoadBalancers(db, common.StringOrNil("region"), nil)

					Expect(lbResult[0]).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"NetworkID": Equal(nlb.ID),
						"Name":      gstruct.PointTo(Equal("LB2")),
						"Region":    gstruct.PointTo(Equal("region")),
						"Type":      gstruct.PointTo(Equal("websocket")),
					})))
					Expect(lbResult).To(HaveLen(1))
					Expect(lbErr).NotTo(HaveOccurred())
				})

				It("should return load balancer with region and type", func() {
					db := dbconf.DatabaseConnection()

					lbResult, lbErr := nlb.LoadBalancers(db, common.StringOrNil("region"), lb2.Type)

					Expect(lbResult[0]).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"NetworkID": Equal(nlb.ID),
						"Name":      gstruct.PointTo(Equal("LB2")),
						"Region":    gstruct.PointTo(Equal("region")),
						"Type":      gstruct.PointTo(Equal("websocket")),
					})))
					Expect(lbResult).To(HaveLen(1))
					Expect(lbErr).NotTo(HaveOccurred())
				})

				It("should return empty load balancer array", func() {
					db := dbconf.DatabaseConnection()

					lbResult, lbErr := nlb.LoadBalancers(db, common.StringOrNil("region"), lb.Type)

					Expect(lbResult).To(HaveLen(0))
					Expect(lbErr).NotTo(HaveOccurred())
				})
			})

			Context("RpcUrl()", func() {
				It("should return LB rpc url", func() {
					Expect(nlb.RpcURL()).To(Equal("url"))
				})

			})

			It("should return status", func() {
				status, statusErr := nlb.Status(true)
				Expect(status).NotTo(BeNil())
				Expect(statusErr).NotTo(HaveOccurred())
			})

		})

		alex */
		Context("with nodes", func() {
			var nwnn *network.Network
			var node *network.NetworkNode
			var runningPeerNode *network.NetworkNode
			var runningFullNode *network.NetworkNode
			var runningValidatorNode *network.NetworkNode
			var runningFaucetNode *network.NetworkNode

			BeforeEach(func() {
				nwnn = &network.Network{
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
					Config: common.MarshalConfig(map[string]interface{}{
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
						"websocket_url":       nil}),
					Stats: nil}

				nwnn.Create()

				nodeConfig := map[string]interface{}{
					"role":     "role",
					"peer_url": "peer_url",
				}
				node = &network.NetworkNode{
					NetworkID: nwnn.ID,
					Bootnode:  true,
					Role:      common.StringOrNil("role"),
					Config:    common.MarshalConfig(nodeConfig),
				}
				r := node.Create()
				fmt.Printf("node created: %t\n", r)

				runningPeerNode = &network.NetworkNode{
					NetworkID: nwnn.ID,
					Bootnode:  true,
					Status:    common.StringOrNil("running"),
					Role:      common.StringOrNil("peer"),
					Config:    common.MarshalConfig(map[string]interface{}{"role": "peer"}),
				}
				runningPeerNode.Create()
				runningFullNode = &network.NetworkNode{
					NetworkID: nwnn.ID,
					Bootnode:  true,
					Status:    common.StringOrNil("running"),
					Role:      common.StringOrNil("full"),
					Config:    common.MarshalConfig(map[string]interface{}{"role": "full"}),
				}
				runningFullNode.Create()
				runningValidatorNode = &network.NetworkNode{
					NetworkID: nwnn.ID,
					Bootnode:  true,
					Status:    common.StringOrNil("running"),
					Role:      common.StringOrNil("validator"),
					Config:    common.MarshalConfig(map[string]interface{}{"role": "validator"}),
				}
				runningValidatorNode.Create()
				runningFaucetNode = &network.NetworkNode{
					NetworkID: nwnn.ID,
					Bootnode:  true,
					Status:    common.StringOrNil("running"),
					Role:      common.StringOrNil("faucet"),
					Config:    common.MarshalConfig(map[string]interface{}{"role": "faucet"}),
				}
				runningFaucetNode.Create()
				time.Sleep(time.Duration(100) * time.Millisecond)
			})

			AfterEach(func() {
				db := dbconf.DatabaseConnection()
				clearNetworks(db)
				db.Delete(network.NetworkNode{})
			})

			It("should return node", func() {
				nodesRes, nodesErr := nwnn.Nodes()
				Expect(nodesRes[0]).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"NetworkID": Equal(nwnn.ID),
					"Role":      gstruct.PointTo(Equal("role")),
				})))
				Expect(nodesRes).To(HaveLen(5))
				Expect(nodesErr).NotTo(HaveOccurred())
			})
			It("should return nodes count", func() {
				Expect(nwnn.BootnodesCount()).To(Equal(uint64(5)))
			})
			It("should return boot nodes txt", func() {
				txt, txtErr := nwnn.BootnodesTxt()
				Expect(txt).To(gstruct.PointTo(Equal("peer_url")))
				Expect(txtErr).NotTo(HaveOccurred())
			})
			It("should return AvailablePeerCount", func() {
				Expect(nwnn.AvailablePeerCount()).To(Equal(uint64(0)))
			})
			It("should return network status", func() {
				Expect(nwnn.Status(true)).NotTo(BeNil())
			})
		})
		Context("without assotiation", func() {
			var nwa *network.Network
			uid_test, _ := uuid.FromString("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
			//var lb, lb2 *network.LoadBalancer
			BeforeEach(func() {

				// ch = make(chan *stan.Msg, 1)
				nwa = &network.Network{
					ApplicationID: nil,
					UserID:        nil,
					Name:          ptrTo("Name ETH non-Cloneable Enabled"),
					Description:   ptrTo("Ethereum Network"),
					IsProduction:  ptrToBool(false),
					Cloneable:     ptrToBool(false),
					Enabled:       ptrToBool(true),
					ChainID:       nil,
					SidechainID:   &uid_test,
					NetworkID:     &uid_test,
					Config: common.MarshalConfig(map[string]interface{}{
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
						"websocket_url":       nil}),
					Stats: nil}
				nwa.Create()

			})
			AfterEach(func() {
				db := dbconf.DatabaseConnection()
				clearNetworks(db)

			})

			Context("reload network ", func() {
				//nlb.Validate()
				first_val_false := false
				first_val_true := true
				uid_test, _ := uuid.FromString("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
				fmt.Println("nwa.ApplicationID >>>>>>>>>>>>>>>>>>>>>>>")
				//fmt.Println(&nwa.ApplicationID)
				//first_uid := nwa.ApplicationID
				//var p *MyError = nil
				It("ApplicationID", func() {
					nwa.ApplicationID = &uid_test
					nwa.Reload()
					Expect(nwa.ApplicationID).To(BeNil())
				})

				It("UserID", func() {
					nwa.UserID = &uid_test
					nwa.Reload()
					Expect(nwa.UserID).To(BeNil())
				})

				It("Name", func() {
					nwa.Name = ptrTo("test")
					nwa.Reload()
					Expect(nwa.Name).To(gstruct.PointTo(Equal("Name ETH non-Cloneable Enabled")))
				})

				It("Description", func() {
					nwa.Description = ptrTo("test")
					nwa.Reload()
					Expect(nwa.Description).To(gstruct.PointTo(Equal("Ethereum Network")))
				})

				It("IsProduction", func() {
					//fmt.Println("IsProduction >>>>>>>>>>>>>>>>")
					//first_uid := nwa.NetworkID
					first_IsProduction := *nwa.IsProduction
					fmt.Println("first_IsProduction")
					fmt.Println(first_IsProduction)
					changed_IsProduction := !first_IsProduction
					fmt.Println("changed_IsProduction")
					fmt.Println(changed_IsProduction)
					nwa.IsProduction = &first_val_true
					nwa.Reload()
					fmt.Println("first_IsProduction")
					fmt.Println(first_IsProduction)
					fmt.Println("nwa.IsProduction")
					fmt.Println(*nwa.IsProduction)
					Expect(*nwa.IsProduction).To(Equal(first_IsProduction))
				})

				It("Cloneable", func() {
					nwa.Cloneable = &first_val_true
					nwa.Reload()
					Expect(*nwa.Cloneable).To(Equal(first_val_false))
				})

				It("Enabled", func() {
					nwa.Enabled = &first_val_false
					nwa.Reload()
					Expect(*nwa.Enabled).To(Equal(first_val_true))
				})
				It("ChainID", func() {
					chid := *nwa.ChainID // being set during Create()
					nwa.ChainID = ptrTo("test")
					nwa.Reload()
					Expect(nwa.ChainID).To(gstruct.PointTo(Equal(chid)))
				})

				It("NetworkID", func() {
					fmt.Println("reload network >>>>>>>>>>>>>>>>")
					first_NetworkID := *nwa.NetworkID
					uid := uuid.Nil
					//uid2, _ := uuid.FromString("1ba7b810-9dad-11d1-80b4-00c04fd430c8")
					nwa.NetworkID = &uid
					//nlb.NetworkID
					nwa.Reload()
					Expect(nwa.NetworkID).To(gstruct.PointTo(Equal(first_NetworkID)))
				})

				It("SidechainID", func() {
					fmt.Println("reload network >>>>>>>>>>>>>>>>")
					first_NetworkID := *nwa.SidechainID
					uid, _ := uuid.FromString("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
					//uid2, _ := uuid.FromString("1ba7b810-9dad-11d1-80b4-00c04fd430c8")
					nwa.SidechainID = &uid
					//nlb.NetworkID
					nwa.Reload()
					Expect(nwa.SidechainID).To(gstruct.PointTo(Equal(first_NetworkID)))
				})

			})

			Context("Update network ", func() {
				first_val_false := false
				first_val_true := true
				uid_test, _ := uuid.FromString("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

				It("ApplicationID", func() {
					nwa.ApplicationID = &uid_test
					nwa.Update()
					Expect(nwa.ApplicationID).To(gstruct.PointTo(Equal(uid_test)))
				})

				It("UserID", func() {
					nwa.UserID = &uid_test
					nwa.Update()
					Expect(*nwa.UserID).To(Equal(uid_test))
				})

				It("Name", func() {
					nwa.Name = ptrTo("test Name")
					nwa.Update()
					Expect(*nwa.Name).To(Equal("test Name"))
				})

				It("Description", func() {
					nwa.Description = ptrTo("test Description")
					nwa.Update()
					Expect(nwa.Description).To(gstruct.PointTo(Equal("test Description")))
				})

				It("IsProduction", func() {
					//fmt.Println("IsProduction >>>>>>>>>>>>>>>>")
					//first_uid := nwa.NetworkID
					first_Update_IsProduction := !*nwa.IsProduction
					nwa.IsProduction = &first_Update_IsProduction
					nwa.Update()
					Expect(*nwa.IsProduction).To(Equal(first_Update_IsProduction))
				})

				It("Cloneable", func() {
					nwa.Cloneable = &first_val_true
					nwa.Update()
					Expect(*nwa.Cloneable).To(Equal(first_val_true))
				})

				It("Enabled", func() {
					nwa.Enabled = &first_val_false
					nwa.Update()
					Expect(*nwa.Enabled).To(Equal(first_val_false))
				})
				It("ChainID", func() {
					nwa.ChainID = ptrTo("test")
					nwa.Update()
					Expect(nwa.ChainID).To(gstruct.PointTo(Equal("test")))
				})

				It("NetworkID", func() {
					fmt.Println("reload network >>>>>>>>>>>>>>>>")
					//first_NetworkID := nwa.NetworkID
					uid := uuid.Nil
					//uid2, _ := uuid.FromString("1ba7b810-9dad-11d1-80b4-00c04fd430c8")
					nwa.NetworkID = &uid
					//nlb.NetworkID
					nwa.Update()
					//Expect(nwa.NetworkID).To(Equal(first_NetworkID))
				})

				It("SidechainID", func() {
					fmt.Println("reload network >>>>>>>>>>>>>>>>")
					//first_NetworkID := nwa.SidechainID
					uid, _ := uuid.FromString("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
					//uid2, _ := uuid.FromString("1ba7b810-9dad-11d1-80b4-00c04fd430c8")
					nwa.SidechainID = &uid
					//nlb.NetworkID
					nwa.Update()
					Expect(*nwa.SidechainID).To(Equal(uid))
				})

			})

		})
	})
})
