package networkfixtures

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/provideapp/goldmine/test/fixtures"
	"github.com/provideapp/goldmine/test/matchers"
)

func ethNonCloneableEnabledChainspecNetwork() (n *fixtures.FixtureMatcher) {
	mc := &matchers.MatcherCollection{}
	optsCreate := defaultMatcherOptions()
	optsCreate["channelPolling"] = true
	optsNATSCreate := defaultNATSMatcherOptions(ptrTo("network.create"))

	mc.AddBehavior("Create", func(opts ...interface{}) types.GomegaMatcher {
		expectedCreateResult := true
		expectedContractCount := 1
		return matchers.NetworkCreateMatcher(expectedCreateResult, expectedContractCount, opts...)
	}, optsCreate)
	mc.AddBehavior("Double Create", func(opts ...interface{}) types.GomegaMatcher {
		return BeFalse()
	}, defaultMatcherOptions())
	mc.AddBehavior("Create with NATS", func(opts ...interface{}) types.GomegaMatcher {
		return matchers.NetworkCreateWithNATSMatcher(true, opts[0].(chan string))
	}, optsNATSCreate)
	mc.AddBehavior("Validate", func(opts ...interface{}) types.GomegaMatcher {
		return BeTrue()
	}, defaultMatcherOptions())
	mc.AddBehavior("ParseConfig", func(opts ...interface{}) types.GomegaMatcher {
		return satisfyAllConfigKeys(false)
	}, defaultMatcherOptions())
	mc.AddBehavior("Network type", func(opts ...interface{}) types.GomegaMatcher {
		if opts[0] == "eth" {
			return BeTrue()
		}
		if opts[0] == "btc" {
			return BeFalse()
		}
		if opts[0] == "handshake" {
			return BeFalse()
		}
		if opts[0] == "ltc" {
			return BeFalse()
		}
		if opts[0] == "quorum" {
			return BeTrue()
		}
		return BeNil()
	}, defaultMatcherOptions())

	name := "ETH NonProd NonClonable Enabled Full Config with Chainspec"
	chainspecJSON, chainspecABIJSON := getChainspec()

	n = &fixtures.FixtureMatcher{
		Fixture: &NetworkFixture{
			Fields: &NetworkFields{
				// ApplicationID: nil,
				// UserID:        nil,
				Name:         ptrTo(name),
				Description:  ptrTo("Ethereum Network"),
				IsProduction: ptrToBool(false),
				Cloneable:    ptrToBool(false),
				Enabled:      ptrToBool(true),
				ChainID:      nil,
				// SidechainID:   nil,
				// NetworkID:     nil,
				Config: marshalConfig(map[string]interface{}{
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
					"websocket_url":       nil}),
				// Stats: nil
			},
			Name: ptrTo(name)},
		Matcher: mc}

	return
}

// func getChainspec() (chainspecJSON map[string]interface{}, chainspecABIJSON map[string]interface{}) {
// 	ethChainspecFileurl := "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn/spec.json"
// 	ethChainspecAbiFileurl := "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json"
// 	response, err := http.Get(ethChainspecFileurl)
// 	//chainspec_text := ""
// 	// chainspec_abi_text := ""
// 	chainspecJSON = map[string]interface{}{}
// 	chainspecABIJSON = map[string]interface{}{}

// 	if err != nil {
// 		fmt.Printf("%s\n", err)
// 	} else {
// 		defer response.Body.Close()
// 		contents, err := ioutil.ReadAll(response.Body)
// 		if err != nil {
// 			fmt.Printf("%s\n", err)
// 		}
// 		// fmt.Printf("%s\n", string(contents))
// 		//chainspec_text = string(contents)
// 		json.Unmarshal(contents, &chainspecJSON)
// 		// common.Log.Debugf("error parsing chainspec: %v", errJSON)

// 	}

// 	responseAbi, err := http.Get(ethChainspecAbiFileurl)

// 	if err != nil {
// 		fmt.Printf("%s\n", err)
// 	} else {
// 		defer responseAbi.Body.Close()
// 		contents, err := ioutil.ReadAll(responseAbi.Body)
// 		if err != nil {
// 			fmt.Printf("%s\n", err)
// 		}
// 		// fmt.Printf("%s\n", string(contents))
// 		// chainspec_abi_text = string(contents)
// 		json.Unmarshal(contents, &chainspecABIJSON)
// 		// common.Log.Debugf("error parsing chainspec: %v", errJSON)
// 	}

// 	return
// }
