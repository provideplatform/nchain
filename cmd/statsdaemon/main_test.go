package main

import (
	"encoding/json"
	"fmt"
	"testing"

	uuid "github.com/kthomas/go.uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/provideplatform/ident/common"
	"github.com/provideplatform/nchain/network"
	ident "github.com/provideplatform/provide-go/api/ident"
	"github.com/provideplatform/provide-go/api/nchain"
)

type chainSpecConfig struct{}

type chainSpec struct {
	Config *chainSpecConfig `json:"config"`
}

type chainConfig struct {
	NativeCurrency      *string    `json:"native_currency"`
	IsBaseledgerNetwork bool       `json:"is_baseledger_network"`
	Client              *string    `json:"client"`
	BlockExplorerUrl    *string    `json:"block_explorer_url"`
	JsonRpcUrl          *string    `json:"json_rpc_url"`
	WebsocketUrl        *string    `json:"websocket_url"`
	Platform            *string    `json:"platform"`
	EngineID            *string    `json:"engine_id"`
	Chain               *string    `json:"chain"`
	ProtocolID          *string    `json:"protocol_id"`
	NetworkID           int        `json:"network_id"`
	ChainSpec           *chainSpec `json:"chainspec"`
}

type chainDef struct {
	Name    *string      `json:"name"`
	Enabled bool         `json:"enabled"`
	Config  *chainConfig `json:"config"`
}

func SetupBaseledgerTestNetwork() (*network.Network, error) {
	testID, _ := uuid.NewV4()

	email := "prvd" + testID.String() + "@email.com"
	pwd := "super_secret"
	_, err := ident.CreateUser("", map[string]interface{}{
		"first_name": "statsdaemon first name" + testID.String(),
		"last_name":  "statsdaemon last name" + testID.String(),
		"email":      email,
		"password":   pwd,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating user. Error: %s", err.Error())
	}

	authResponse, _ := ident.Authenticate(email, pwd)
	if err != nil {
		return nil, fmt.Errorf("error authenticating user. Error: %s", err.Error())
	}

	chainySpecConfig := chainSpecConfig{}
	chainySpec := chainSpec{
		Config: &chainySpecConfig,
	}
	chainyConfig := chainConfig{
		NativeCurrency:      common.StringOrNil("token"),
		Platform:            common.StringOrNil("tendermint"),
		Client:              common.StringOrNil("baseledger"),
		NetworkID:           3,
		IsBaseledgerNetwork: true,
		Chain:               common.StringOrNil("peachtree"),
		WebsocketUrl:        common.StringOrNil("ws://genesis.peachtree.baseledger.provide.network:1337/websocket"),
		ChainSpec:           &chainySpec,
	}

	chainName := fmt.Sprintf("Baseledger Testnet %s", testID.String())

	chainyChain := chainDef{
		Name:   common.StringOrNil(chainName),
		Config: &chainyConfig,
	}

	chainyChainJSON, _ := json.Marshal(chainyChain)

	params := map[string]interface{}{}
	json.Unmarshal(chainyChainJSON, &params)

	testNetwork, err := nchain.CreateNetwork(*authResponse.Token.AccessToken, params)
	if err != nil {
		return nil, fmt.Errorf("error creating network. Error: %s", err.Error())
	}

	return &network.Network{
		ApplicationID: testNetwork.ApplicationID,
		UserID:        testNetwork.UserID,
		Name:          testNetwork.Name,
		Description:   testNetwork.Description,
		Enabled:       testNetwork.Enabled,
		ChainID:       testNetwork.ChainID,
		NetworkID:     testNetwork.NetworkID,
		Config:        testNetwork.Config,
	}, nil
}

func TestNChainBaseledgerStatsdaemon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NChain statsdaemon Suite")

	testNetwork, err := SetupBaseledgerTestNetwork()
	if err != nil {
		t.Errorf("Failed to set up baseledger test network %v", err.Error())
	}

	statsDaemon := NewNetworkStatsDaemon(common.Log, testNetwork)

	testCh := make(chan *nchain.NetworkStatus)
	// go routine since this is blocking
	go statsDaemon.dataSource.Stream(testCh)
	// get one result and shutdown statsdaemon and check result
	sampleResult := <-testCh
	statsDaemon.shutdown()

	jsonSampleResult, _ := json.Marshal(sampleResult.Meta["last_block_header"])
	formattedSampleHeaderResult := nchain.BaseledgerBlockHeaderResponse{}.Value.Header
	err = json.Unmarshal(jsonSampleResult, &formattedSampleHeaderResult)
	if err != nil {
		t.Errorf("Failed to unmarshall header response %v", err.Error())
	}

	t.Logf("Header response success %v\n", formattedSampleHeaderResult)
}

var _ = Describe("Main", func() {

})
