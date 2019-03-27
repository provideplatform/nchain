package network

import (
	"fmt"

	"github.com/provideapp/goldmine/test/matchers"

	dbconf "github.com/kthomas/go-db-config"
	stan "github.com/nats-io/go-nats-streaming"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Network", func() {
	var n *Network
	var mc *matchers.MatcherCollection
	var ch chan *stan.Msg
	// var chStr chan string
	var natsConn stan.Conn
	var natsSub stan.Subscription
	var err error

	var chPolling chan string

	var networks, _ = testNetworks()

	BeforeEach(func() {

		n = &Network{
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

		ch = make(chan *stan.Msg, 1)

	})

	AfterEach(func() {

		db := dbconf.DatabaseConnection()
		db.Delete(Network{})
		db.Delete(Contract{})

		if natsSub != nil {
			natsSub.Unsubscribe()
		}
	})

	Describe("Network", func() {
		Context("production", func() {})

		Context("network fixtures", func() {
			It("should cover all generator cases", func() {
				// fixtures := networkFixtureGenerator.All()
				// Expect(len(fixtures) - len(networks)).To(Equal(0))
				// Expect(fixtures).To(HaveLen(8))
				// FIXME: Expect(rest).To(HaveLen(0))
			})
		})

		// TODO:
		//   1. add mocks to check NATS and other calls (we can't just check all NATS channels to see nothing is written)
		//   2. add behaviors for private methods
		//   3. add config keys to generator
		Context("Dynamic", func() {

			for i := 0; i < len(networks); i++ {

				nn := networks[i] // current network being tested
				name := *nn.name  // current network name

				Context(name, func() { // context for current network

					BeforeEach(func() {
						n = nn.network() // creating new pointer with network data for each test
						mc = nn.matchers // set of matchers for current network
					})

					Context("NATS", func() {
						BeforeEach(func() {

							matcherName := "Create with NATS"
							var chName string
							if opts, ok := mc.MatcherOptionsFor(matcherName); ok {
								chName = *opts.NATSChannels[0]
							}

							natsConn = getNatsStreamingConnection()
							ch = make(chan *stan.Msg, 1)
							natsSub, err = natsConn.QueueSubscribe(chName, chName, func(msg *stan.Msg) {
								ch <- msg
							})
							if err != nil {
								Log.Debugf("conn failure")
							}

							natsGuaranteeDelivery(chName)
						})
						It("should catch NATS message", func() {
							chPolling = make(chan string, 1)
							cf := func(ch chan string) error {
								return nil
							}
							pollingToStrChFunc(chPolling, cf, nil)

							matcherName := "Create with NATS"
							Expect(n.Create()).To(mc.MatchBehaviorFor(matcherName, chPolling))
						})
					})

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

								objects := []Contract{}
								db.Find(&objects)

								for _, object := range objects {
									fmt.Println(object.ID.String())
									ch <- object.ID.String()
								}

								return nil
							}

							pollingToStrChFunc(chPolling, cf, nil) // last param nil to receive default message "timeout"
							// 	}
							// }

							funcAfter = func() []interface{} {
								objects := []Contract{}
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
						Expect(n.Validate()).To(mc.MatchBehaviorFor("Validate"))
					})
					It("should parse config", func() {
						Expect(n.ParseConfig()).To(mc.MatchBehaviorFor("ParseConfig"))
					})
					It("should return network type correctly", func() {
						Expect(n.isEthereumNetwork()).To(mc.MatchBehaviorFor("Network type", "eth"))
						Expect(n.isBcoinNetwork()).To(mc.MatchBehaviorFor("Network type", "btc"))
						Expect(n.isHandshakeNetwork()).To(mc.MatchBehaviorFor("Network type", "handshake"))
						Expect(n.isLcoinNetwork()).To(mc.MatchBehaviorFor("Network type", "ltc"))
						Expect(n.isQuorumNetwork()).To(mc.MatchBehaviorFor("Network type", "quorum"))
					})
					It("should not create second record", func() {
						n.Create()
						Expect(n.Create()).To(mc.MatchBehaviorFor("Double Create"))
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
						// Expect(n.NodeCount()).To(mc.MatchBehaviorFor("NodeCount"))
					})
					It("should return AvailablePeerCount", func() {
						// Expect(n.AvailablePeerCount()).To(mc.MatchBehaviorFor("AvailablePeerCount"))
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

	})
})
