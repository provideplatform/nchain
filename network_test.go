package main

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

func ptrTo(s string) *string {
	return &s
}

func TestNetwork_Create(t *testing.T) {
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
		{"when ETH network is valid",
			fields{
				ApplicationID: nil,
				UserID:        nil,
				Name:          StringOrNil("Name"),
				Description:   StringOrNil("Description"),
				IsProduction:  boolOrNil(false),
				Cloneable:     boolOrNil(true),
				Enabled:       boolOrNil(true),
				ChainID:       nil,
				SidechainID:   nil,
				NetworkID:     nil,
				Config: marshalConfig(map[string]interface{}{
					"block_explorer_url": "https://unicorn-explorer.provide.network",
					"chain":              "unicorn-v0",
					"chainspec_abi_url":  "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
					"chainspec_url":      "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json",
					"cloneable_cfg": map[string]interface{}{
						"_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}},
						"aws":       map[string]interface{}{"docker": map[string]interface{}{"regions": map[string]interface{}{"ap-northeast-1": map[string]interface{}{"peer": "providenetwork-node", "validator": "providenetwork-node"}}}}},
					"engine_id":           "authorityRound",
					"is_ethereum_network": true,
					"is_load_balanced":    true,
					"json_rpc_url":        "http://ec2-54-167-229-155.compute-1.amazonaws.com:8050",
					"native_currency":     "PRVD",
					"network_id":          22,
					"protocol_id":         "poa",
					"websocket_url":       "ws://ec2-54-167-229-155.compute-1.amazonaws.com:8051"}),
				Stats: nil},
			true},
		{"when config is empty",
			fields{
				Name:         StringOrNil("Name2"),
				Description:  StringOrNil("Description2"),
				IsProduction: boolOrNil(false),
				Cloneable:    boolOrNil(false),
				Enabled:      boolOrNil(false),
				Config:       nil,
			},
			false}}
	// mockCtrl := gomock.NewController(t)
	// defer mockCtrl.Finish()

	// mock := mocks.NewMockIModel(mockCtrl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Network{
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
				res2B, _ := json.Marshal(n)
				networkID, _ := hexutil.DecodeBig(*n.ChainID)
				t.Errorf(
					"Network.Create() = %v, want %v; chain_id: %v, network_id: %v, network: %v",
					got,
					tt.want,
					*n.ChainID,
					networkID,
					string(res2B))
			}
		})
	}
}
