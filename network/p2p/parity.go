package p2p

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

// ParityP2PProvider is a network.p2p.API implementing the parity API
type ParityP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
}

// InitParityP2PProvider initializes and returns the parity p2p provider
func InitParityP2PProvider(rpcURL *string, ntwrk common.Configurable) *ParityP2PProvider {
	return &ParityP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
	}
}

// DefaultEntrypoint returns the default entrypoint to run when starting the container, when one is not otherwise provided
func (p *ParityP2PProvider) DefaultEntrypoint() []string {
	return []string{
		"parity",
		"--loging", "verbose",
		"--fat-db", "on",
		"--pruning", "archive",
		"--tracing", "on",
		"--jsonrpc-apis", "web3,eth,net,personal,parity,parity_set,traces,rpc,parity_accounts",
		"--jsonrpc-hosts", "all",
		"--jsonrpc-interface", "0.0.0.0",
		"--jsonrpc-cors", "all",
		"--tx-queue-per-sender", "2048",
		"--ws-apis", "web3,eth,pubsub,net,parity,parity_pubsub,traces,rpc,shh,shh_pubsub",
		"--ws-hosts", "all",
		"--ws-interface", "0.0.0.0",
		"--ws-origins", "all",
		"--ws-max-connections", "2048",
	}
}

// EnrichStartCommand returns the cmd to append to the command to start the container
func (p *ParityP2PProvider) EnrichStartCommand(bootnodes []string) []string {
	cmd := make([]string, 0)
	cfg := p.network.ParseConfig()
	if networkID, networkIDOk := cfg["network_id"].(float64); networkIDOk {
		cmd = append(cmd, "--chain")
		cmd = append(cmd, fmt.Sprintf("%f", networkID))
	}

	cfgBootnodes, cfgBootnodesOk := cfg["bootnodes"].([]string)
	if len(bootnodes) > 0 || (cfgBootnodesOk && len(cfgBootnodes) > 0) {
		_bootnodes := make([]string, 0)
		for i := range bootnodes {
			_bootnodes = append(_bootnodes, bootnodes[i])
		}
		if cfgBootnodesOk {
			for i := range cfgBootnodes {
				_bootnodes = append(_bootnodes, cfgBootnodes[i])
			}
		}
		cmd = append(cmd, "--bootnodes", p.FormatBootnodes(_bootnodes))
	}

	encryptedCfg, _ := p.network.DecryptedConfig()
	if env, envOk := encryptedCfg["env"].(map[string]interface{}); envOk {
		engineSigner, engineSignerOk := env["ENGINE_SIGNER"].(string)
		engineSignerKeyJSON, engineSignerKeyJSONOk := env["ENGINE_SIGNER_KEY_JSON"].(string)
		engineSignerPrivateKey, engineSignerPrivateKeyOk := env["ENGINE_SIGNER_PRIVATE_KEY"].(string)

		if engineSignerOk && engineSignerKeyJSONOk && engineSignerPrivateKeyOk {
			// FIXME-- modify the entrypoint to write the --password file
			cmd = append(cmd, "--engine-signer")
			cmd = append(cmd, fmt.Sprintf("--engine-signer %s", engineSigner))

			cmd = append(cmd, "--password")
			cmd = append(cmd, fmt.Sprintf("--password %s", engineSignerKeyJSON))

			cmd = append(cmd, "--author")
			cmd = append(cmd, fmt.Sprintf("%s", engineSignerPrivateKey))
		}
	}

	return cmd
}

// AcceptNonReservedPeers allows non-reserved peers to connect
func (p *ParityP2PProvider) AcceptNonReservedPeers() error {
	var resp interface{}
	return provide.EVMInvokeJsonRpcClient(*p.rpcClientKey, *p.rpcURL, "parity_acceptNonReservedPeers", []interface{}{}, &resp)
}

// DropNonReservedPeers only allows reserved peers to connect; reversed by calling `AcceptNonReservedPeers`
func (p *ParityP2PProvider) DropNonReservedPeers() error {
	var resp interface{}
	return provide.EVMInvokeJsonRpcClient(*p.rpcClientKey, *p.rpcURL, "parity_dropNonReservedPeers", []interface{}{}, &resp)
}

// AddPeer adds a peer by its peer url
func (p *ParityP2PProvider) AddPeer(peerURL string) error {
	var resp interface{}
	return provide.EVMInvokeJsonRpcClient(*p.rpcClientKey, *p.rpcURL, "parity_addReservedPeer", []interface{}{peerURL}, &resp)
}

// FormatBootnodes formats the given peer urls as a valid bootnodes param
func (p *ParityP2PProvider) FormatBootnodes(bootnodes []string) string {
	return strings.Join(bootnodes, ",")
}

// ParsePeerURL parses a peer url from the given raw log string
func (p *ParityP2PProvider) ParsePeerURL(string) (*string, error) {
	return nil, errors.New("parity p2p provider does not impl ParsePeerURL()")
}

// RemovePeer removes a peer by its peer url
func (p *ParityP2PProvider) RemovePeer(peerURL string) error {
	var resp interface{}
	return provide.EVMInvokeJsonRpcClient(*p.rpcClientKey, *p.rpcURL, "parity_removeReservedPeer", []interface{}{peerURL}, &resp)
}

// ResolvePeerURL attempts to resolve one or more viable peer urls
func (p *ParityP2PProvider) ResolvePeerURL() (*string, error) {
	var resp interface{}
	err := provide.EVMInvokeJsonRpcClient(*p.rpcClientKey, *p.rpcURL, "parity_enode", []interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	if response, responseOk := resp.(map[string]interface{}); responseOk {
		if peerURL, peerURLOk := response["result"].(string); peerURLOk {
			return &peerURL, nil
		}
	}
	return nil, errors.New("Failed to resolve peer url for parity_enode json-rpc response")
}

// RequireBootnodes attempts to resolve the peers to use as bootnodes
func (p *ParityP2PProvider) RequireBootnodes(db *gorm.DB, userID *uuid.UUID, networkID *uuid.UUID, n common.Configurable) error {
	var err error

	cfg := p.network.ParseConfig()
	encryptedCfg, err := n.DecryptedConfig()
	if err != nil {
		return fmt.Errorf("Failed to decrypt config for network node: %s", err.Error())
	}
	env, envOk := cfg["env"].(map[string]interface{})
	encryptedEnv, encryptedEnvOk := encryptedCfg["env"].(map[string]interface{})

	if envOk && encryptedEnvOk {
		var addr *string
		var privateKey *ecdsa.PrivateKey
		_, masterOfCeremonyPrivateKeyOk := encryptedEnv["ENGINE_SIGNER_PRIVATE_KEY"].(string)
		if masterOfCeremony, masterOfCeremonyOk := env["ENGINE_SIGNER"].(string); masterOfCeremonyOk && !masterOfCeremonyPrivateKeyOk {
			addr = common.StringOrNil(masterOfCeremony)
			out := []string{}
			db.Table("accounts").Select("private_key").Where("accounts.user_id = ? AND accounts.address = ?", userID.String(), addr).Pluck("private_key", &out)
			if out == nil || len(out) == 0 || len(out[0]) == 0 {
				common.Log.Warningf("Failed to retrieve manage engine signing identity for network: %s; generating unmanaged identity...", networkID)
				addr, privateKey, err = provide.EVMGenerateKeyPair()
			} else {
				encryptedKey := common.StringOrNil(out[0])
				privateKey, err = common.DecryptECDSAPrivateKey(*encryptedKey)
				if err == nil {
					common.Log.Debugf("Decrypted private key for master of ceremony: %s", *addr)
				} else {
					msg := fmt.Sprintf("Failed to decrypt private key for master of ceremony on network: %s", networkID)
					common.Log.Warning(msg)
					return errors.New(msg)
				}
			}
		} else if !masterOfCeremonyPrivateKeyOk {
			common.Log.Debugf("Generating managed master of ceremony signing identity for network: %s", networkID)
			addr, privateKey, err = provide.EVMGenerateKeyPair()
		}

		if addr != nil && privateKey != nil {
			keystoreJSON, err := provide.EVMMarshalEncryptedKey(ethcommon.HexToAddress(*addr), privateKey, hex.EncodeToString(ethcrypto.FromECDSA(privateKey)))
			if err == nil {
				common.Log.Debugf("Master of ceremony has initiated the initial key ceremony: %s; network: %s", *addr, networkID)
				env["ENGINE_SIGNER"] = addr
				encryptedEnv["ENGINE_SIGNER_PRIVATE_KEY"] = hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
				encryptedEnv["ENGINE_SIGNER_KEY_JSON"] = string(keystoreJSON)

				n.SetConfig(cfg)
				n.SetEncryptedConfig(encryptedCfg)
				n.SanitizeConfig()
				db.Save(&n)

				networkCfg := p.network.ParseConfig()
				if chainspec, chainspecOk := networkCfg["chainspec"].(map[string]interface{}); chainspecOk {
					if accounts, accountsOk := chainspec["accounts"].(map[string]interface{}); accountsOk {
						nonSystemAccounts := make([]string, 0)
						for account := range accounts {
							if !strings.HasPrefix(account, "0x000000000000000000000000000000000") { // 7 chars truncated
								nonSystemAccounts = append(nonSystemAccounts, account)
							}
						}
						if len(nonSystemAccounts) == 1 {
							templateMasterOfCeremony := nonSystemAccounts[0]
							chainspecJSON, err := json.Marshal(chainspec)
							if err == nil {
								chainspecJSON = []byte(strings.Replace(string(chainspecJSON), templateMasterOfCeremony[2:], string(*addr)[2:], -1))
								chainspecJSON = []byte(strings.Replace(string(chainspecJSON), strings.ToLower(templateMasterOfCeremony[2:]), strings.ToLower(string(*addr)[2:]), -1))
								var newChainspec map[string]interface{}
								err = json.Unmarshal(chainspecJSON, &newChainspec)
								if err == nil {
									networkCfg["chainspec"] = newChainspec
									p.network.SetConfig(networkCfg)
									db.Save(&p.network)
								}
							}
						}
					}
				}
			} else {
				common.Log.Warningf("Failed to generate master of ceremony address for network: %s; %s", networkID, err.Error())
			}
		}
	}

	return err
}

// Upgrade executes a pending upgrade
func (p *ParityP2PProvider) Upgrade() error {
	var resp interface{}
	return provide.EVMInvokeJsonRpcClient(*p.rpcClientKey, *p.rpcURL, "parity_executeUpgrade", []interface{}{}, &resp)
}
