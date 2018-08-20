package main

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/kthomas/go.uuid"
	"github.com/provideapp/go-core"
)

func TestContract_Execute(t *testing.T) {
	type fields struct {
		Model         gocore.Model
		ApplicationID *uuid.UUID
		NetworkID     uuid.UUID
		TransactionID *uuid.UUID
		Name          *string
		Address       *string
		Params        *json.RawMessage
		AccessedAt    *time.Time
	}
	type args struct {
		wallet *Wallet
		value  *big.Int
		method string
		params []interface{}
		gas    uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ContractExecutionResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Contract{
				Model:         tt.fields.Model,
				ApplicationID: tt.fields.ApplicationID,
				NetworkID:     tt.fields.NetworkID,
				TransactionID: tt.fields.TransactionID,
				Name:          tt.fields.Name,
				Address:       tt.fields.Address,
				Params:        tt.fields.Params,
				AccessedAt:    tt.fields.AccessedAt,
			}
			got, err := c.Execute(tt.args.wallet, tt.args.value, tt.args.method, tt.args.params, tt.args.gas)
			if (err != nil) != tt.wantErr {
				t.Errorf("Contract.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Contract.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
