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
	"fmt"

	"github.com/provideplatform/nchain/common"
)

const bcoinPoolIdentitySearchString = "Pool identity key:"

// BcoinP2PProvider is a network.p2p.API implementing the Bcoin API
type BcoinP2PProvider struct {
	rpcClientKey string
	rpcURL       string
}

// InitBcoinP2PProvider initializes and returns the bcoin p2p provider
func InitBcoinP2PProvider(host string, port uint, ntwrk common.Configurable) *BcoinP2PProvider {
	return nil
}

// ParsePeerURL parses the peer url from the given logs
func (p *BcoinP2PProvider) ParsePeerURL(msg string) (*string, error) {
	// poolIdentityFoundIndex := strings.LastIndex(msg, bcoinPoolIdentitySearchString)
	// if poolIdentityFoundIndex != -1 {
	// 	defaultPeerListenPort := common.DefaultPeerDiscoveryPort
	// 	poolIdentity := strings.TrimSpace(msg[poolIdentityFoundIndex+len(bcoinPoolIdentitySearchString) : len(msg)-1])
	// 	node := fmt.Sprintf("%s@%s:%v", poolIdentity, *n.IPv4, defaultPeerListenPort)
	// 	peerURL = &node
	// 	cfg["peer_url"] = node
	// 	cfg["peer_identity"] = poolIdentity
	// }

	return nil, fmt.Errorf("bcoin p2p provider does not yet implement ParsePeerURL(")
}
