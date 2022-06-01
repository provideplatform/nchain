/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package p2p

import (
	"errors"
	"math/big"
	"strings"

	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	provide "github.com/provideplatform/provide-go/api/nchain"
)

// BaseledgerP2PProvider is a network.p2p.API implementing the baseledger API
type BaseledgerP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
	networkID    string
}

// BaseledgerP2PProvider initializes and returns the baseledger p2p provider
func InitBaseledgerP2PProvider(rpcURL *string, networkID string, ntwrk common.Configurable) *BaseledgerP2PProvider {
	return &BaseledgerP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
		networkID:    networkID,
	}
}

// DefaultEntrypoint returns the default entrypoint to run when starting the container, when one is not otherwise provided
func (p *BaseledgerP2PProvider) DefaultEntrypoint() []string {
	return []string{}
}

// EnrichStartCommand returns the cmd to append to the command to start the container
func (p *BaseledgerP2PProvider) EnrichStartCommand(bootnodes []string) []string {
	cmd := make([]string, 0)

	return cmd
}

// AcceptNonReservedPeers allows non-reserved peers to connect
func (p *BaseledgerP2PProvider) AcceptNonReservedPeers() error {
	return errors.New("not yet implemented")
}

// DropNonReservedPeers only allows reserved peers to connect; reversed by calling `AcceptNonReservedPeers`
func (p *BaseledgerP2PProvider) DropNonReservedPeers() error {
	return errors.New("not yet implemented")
}

// AddPeer adds a peer by its peer url
func (p *BaseledgerP2PProvider) AddPeer(peerURL string) error {
	return errors.New("not yet implemented")
}

// FetchTxReceipt fetch a transaction receipt given its hash
func (p *BaseledgerP2PProvider) FetchTxReceipt(signerAddress, hash string) (*provide.TxReceipt, error) {
	return nil, errors.New("not yet implemented")
}

// FetchTxTraces fetch transaction traces given its hash
func (p *BaseledgerP2PProvider) FetchTxTraces(hash string) (*provide.TxTrace, error) {
	return nil, errors.New("not yet implemented")
}

// FormatBootnodes formats the given peer urls as a valid bootnodes param
func (p *BaseledgerP2PProvider) FormatBootnodes(bootnodes []string) string {
	return strings.Join(bootnodes, ",")
}

// ParsePeerURL parses a peer url from the given raw log string
func (p *BaseledgerP2PProvider) ParsePeerURL(string) (*string, error) {
	return nil, errors.New("not yet implemented")
}

// RemovePeer removes a peer by its peer url
func (p *BaseledgerP2PProvider) RemovePeer(peerURL string) error {
	return errors.New("not yet implemented")
}

// ResolvePeerURL attempts to resolve one or more viable peer urls
func (p *BaseledgerP2PProvider) ResolvePeerURL() (*string, error) {
	return nil, errors.New("not yet implemented")
}

// ResolveTokenContract attempts to resolve the given token contract details for the contract at a given address
func (p *BaseledgerP2PProvider) ResolveTokenContract(signerAddress string, receipt interface{}, artifact *provide.CompiledArtifact) (*string, *string, *big.Int, *string, error) {
	return nil, nil, nil, nil, errors.New("not yet implemented")
}

// RequireBootnodes attempts to resolve the peers to use as bootnodes
func (p *BaseledgerP2PProvider) RequireBootnodes(db *gorm.DB, userID *uuid.UUID, networkID *uuid.UUID, n common.Configurable) error {
	return errors.New("not yet implemented")
}

// Upgrade executes a pending upgrade
func (p *BaseledgerP2PProvider) Upgrade() error {
	return errors.New("not yet implemented")
}
