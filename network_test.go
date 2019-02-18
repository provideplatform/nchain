package main

import (
	"encoding/json"
	"testing"

	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

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
		// TODO: Add test cases.
	}
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
				t.Errorf("Network.Create() = %v, want %v", got, tt.want)
			}
		})
	}
}
