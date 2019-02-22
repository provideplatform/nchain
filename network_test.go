package main_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provideapp "github.com/provideapp/goldmine"
	provide "github.com/provideservices/provide-go"
)

func ptrTo(s string) *string {
	return &s
}
func ptrToBool(s bool) *bool {
	return &s
}
func marshalConfig(opts map[string]interface{}) *json.RawMessage {
	cfgJSON, _ := json.Marshal(opts)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

func setupTestCase(t *testing.T) func(t *testing.T) {
	t.Log("setup test case")
	return func(t *testing.T) {
		t.Log("teardown test case: removing networks")
		db := dbconf.DatabaseConnection()
		db.Delete(provideapp.Network{})
	}
}

func TestNetwork_Create(t *testing.T) {

	eth_chainspec_fileurl := "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn/spec.json"
	eth_chainspec_abi_fileurl := "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json"
	response, err := http.Get(eth_chainspec_fileurl)
	chainspec_text := ""
	chainspec_abi_text := ""

	if err != nil {
		fmt.Printf("%s\n", err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		// fmt.Printf("%s\n", string(contents))
		chainspec_text = string(contents)
	}

	response_abi, err := http.Get(eth_chainspec_abi_fileurl)

	if err != nil {
		fmt.Printf("%s\n", err)
	} else {
		defer response_abi.Body.Close()
		contents, err := ioutil.ReadAll(response_abi.Body)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		// fmt.Printf("%s\n", string(contents))
		chainspec_abi_text = string(contents)
	}

	type fields struct {
		Model         provide.Model
		ApplicationID *uuid.UUID
		UserID        *uuid.UUID
		Name          *string
		Description   *string
		IsProduction  *bool
		Cloneable     *bool
		Enabled       *bool
		ChainID       *string
		SidechainID   *uuid.UUID
		NetworkID     *uuid.UUID
		Config        *json.RawMessage
		Stats         *provide.NetworkStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"when ETH network is not clonable, enabled, and valid with chainspec url",
			fields{
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
				Stats: nil},
			true},
		{"when ETH network is not clonable, enabled, and valid with raw chainspec",
			fields{
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
					"block_explorer_url":  "https://unicorn-explorer.provide.network",
					"chain":               "unicorn-v0",
					"chainspec":           chainspec_text,
					"chainspec_abi":       chainspec_abi_text,
					"cloneable_cfg":       map[string]interface{}{},
					"engine_id":           "authorityRound", // required
					"is_ethereum_network": true,
					"is_load_balanced":    false,
					"json_rpc_url":        nil,
					"native_currency":     "PRVD", // required
					"network_id":          22,
					"protocol_id":         "poa", // required
					"websocket_url":       nil}),
				Stats: nil},
			true},
		{"when ETH network is clonable, enabled, and valid with chainspec url",
			fields{
				ApplicationID: nil,
				UserID:        nil,
				Name:          ptrTo("Name ETH Cloneable Enabled"),
				Description:   ptrTo("Ethereum Network cloneable"),
				IsProduction:  ptrToBool(false),
				Cloneable:     ptrToBool(true),
				Enabled:       ptrToBool(true),
				ChainID:       nil,
				SidechainID:   nil,
				NetworkID:     nil,
				Config: marshalConfig(map[string]interface{}{
					"block_explorer_url": "https://unicorn-explorer.provide.network",
					"chain":              "unicorn-v0",
					"chainspec_abi_url":  "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
					"chainspec_url":      "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json",
					"cloneable_cfg": map[string]interface{}{
						"_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security
					"engine_id":           "authorityRound",
					"is_ethereum_network": true,
					"is_load_balanced":    false,
					"json_rpc_url":        nil,
					"native_currency":     "PRVD",
					"network_id":          22,
					"protocol_id":         "poa",
					"websocket_url":       nil}),
				Stats: nil},
			true},
		{"when ETH network is clonable, enabled, and not valid because of missing security config",
			fields{
				ApplicationID: nil,
				UserID:        nil,
				Name:          ptrTo("Name ETH Cloneable Enabled"),
				Description:   ptrTo("Ethereum Network cloneable"),
				IsProduction:  ptrToBool(false),
				Cloneable:     ptrToBool(true),
				Enabled:       ptrToBool(true),
				ChainID:       nil,
				SidechainID:   nil,
				NetworkID:     nil,
				Config: marshalConfig(map[string]interface{}{
					"block_explorer_url":  "https://unicorn-explorer.provide.network",
					"chain":               "unicorn-v0",
					"chainspec_abi_url":   "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
					"chainspec_url":       "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json",
					"cloneable_cfg":       map[string]interface{}{}, // If cloneable CFG then security
					"engine_id":           "authorityRound",
					"is_ethereum_network": true,
					"is_load_balanced":    false,
					"json_rpc_url":        nil,
					"native_currency":     "PRVD",
					"network_id":          22,
					"protocol_id":         "poa",
					"websocket_url":       nil}),
				Stats: nil},
			false},
		{"when ETH network is clonable, not enabled, and valid with chainspec url",
			fields{
				ApplicationID: nil,
				UserID:        nil,
				Name:          ptrTo("Name ETH Cloneable Disabled"),
				Description:   ptrTo("Ethereum Network cloneable"),
				IsProduction:  ptrToBool(false),
				Cloneable:     ptrToBool(true),
				Enabled:       ptrToBool(false),
				ChainID:       nil,
				SidechainID:   nil,
				NetworkID:     nil,
				Config: marshalConfig(map[string]interface{}{
					"block_explorer_url": "https://unicorn-explorer.provide.network",
					"chain":              "unicorn-v0",
					"chainspec_abi_url":  "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
					"chainspec_url":      "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json",
					"cloneable_cfg": map[string]interface{}{
						"_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security
					"engine_id":           "authorityRound",
					"is_ethereum_network": true,
					"is_load_balanced":    false,
					"json_rpc_url":        nil,
					"native_currency":     "PRVD",
					"network_id":          22,
					"protocol_id":         "poa",
					"websocket_url":       nil}),
				Stats: nil},
			true},
		{"when ETH network is not clonable, enabled, and not valid because of empty config",
			fields{
				ApplicationID: nil,
				UserID:        nil,
				Name:          ptrTo("Name ETH non-Cloneable Enabled empty Config"),
				Description:   ptrTo("Ethereum Network"),
				IsProduction:  ptrToBool(false),
				Cloneable:     ptrToBool(false),
				Enabled:       ptrToBool(true),
				ChainID:       nil,
				SidechainID:   nil,
				NetworkID:     nil,
				Config:        marshalConfig(map[string]interface{}{}),
				Stats:         nil},
			false},
		{"when ETH network is not clonable, disabled, and valid despite of empty config",
			fields{
				ApplicationID: nil,
				UserID:        nil,
				Name:          ptrTo("Name ETH non-Cloneable Disabled"),
				Description:   ptrTo("Ethereum Network"),
				IsProduction:  ptrToBool(false),
				Cloneable:     ptrToBool(false),
				Enabled:       ptrToBool(false),
				ChainID:       nil,
				SidechainID:   nil,
				NetworkID:     nil,
				Config:        marshalConfig(map[string]interface{}{}),
				Stats:         nil},
			true},
		{"when BTC network is not valid",
			fields{
				ApplicationID: nil,
				UserID:        nil,
				Name:          ptrTo("Name BTC"),
				Description:   ptrTo("Bitcoin Network"),
				IsProduction:  ptrToBool(false),
				Cloneable:     ptrToBool(false),
				Enabled:       ptrToBool(false),
				ChainID:       nil,
				SidechainID:   nil,
				NetworkID:     nil,
				Config:        marshalConfig(map[string]interface{}{}),
				Stats:         nil},
			false},
		{"when clonable is nil is not valid",
			fields{
				Name:         ptrTo("Name Clonable nil"),
				Description:  ptrTo("Description2"),
				IsProduction: ptrToBool(false),
				Cloneable:    nil,
				Enabled:      ptrToBool(false),
				Config:       nil,
			},
			false},
		{"when clonable is nil is not valid",
			fields{
				Name:         ptrTo("Name Enabled nil"),
				Description:  ptrTo("Description2"),
				IsProduction: ptrToBool(false),
				Cloneable:    ptrToBool(false),
				Enabled:      nil,
				Config:       nil,
			},
			false},
		{"when clonable is nil is not valid",
			fields{
				Name:         ptrTo("Name isProduction nil"),
				Description:  ptrTo("Description2"),
				IsProduction: nil,
				Cloneable:    ptrToBool(false),
				Enabled:      ptrToBool(false),
				Config:       nil,
			},
			false},
		{"when config is nil is not valid",
			fields{
				Name:         ptrTo("Name2"),
				Description:  ptrTo("Description2"),
				IsProduction: ptrToBool(false),
				Cloneable:    ptrToBool(false),
				Enabled:      ptrToBool(false),
				Config:       nil,
			},
			false}}
	// mockCtrl := gomock.NewController(t)
	// defer mockCtrl.Finish()

	// mock := mocks.NewMockIModel(mockCtrl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teardownTestCase := setupTestCase(t)
			defer teardownTestCase(t)

			n := &provideapp.Network{
				Model:         tt.fields.Model,
				ApplicationID: tt.fields.ApplicationID,
				UserID:        tt.fields.UserID,
				Name:          tt.fields.Name,
				Description:   tt.fields.Description,
				IsProduction:  tt.fields.IsProduction,
				Cloneable:     tt.fields.Cloneable,
				Enabled:       tt.fields.Enabled,
				ChainID:       tt.fields.ChainID,
				SidechainID:   tt.fields.SidechainID,
				NetworkID:     tt.fields.NetworkID,
				Config:        tt.fields.Config,
				Stats:         tt.fields.Stats,
			}
			if got := n.Create(); got != tt.want {
				// res2B, _ := json.Marshal(n)
				// networkID, _ := hexutil.DecodeBig(*n.ChainID)
				t.Errorf(
					"Network.Create() = %v, want %v; network: %v",
					got,
					tt.want,
					n)
				// string(res2B))
			}
		})
	}
}
