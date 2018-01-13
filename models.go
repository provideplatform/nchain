package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	ethparams "github.com/ethereum/go-ethereum/params"

	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
)

type Model struct {
	Id        uuid.UUID `sql:"primary_key;type:uuid;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time `sql:"not null;default:now()" json:"created_at"`
	Errors    []*Error  `gorm:"-" json:"-"`
}

type Error struct {
	Message *string `json:"message"`
	Status  *int    `json:"status"`
}

type Network struct {
	Model
	Name         *string          `sql:"not null" json:"name"`
	Description  *string          `json:"description"`
	IsProduction *bool            `sql:"not null" json:"is_production"`
	SidechainId  *uuid.UUID       `sql:"type:uuid" json:"sidechain_id"` // network id used as the transactional sidechain (or null)
	Config       *json.RawMessage `sql:"type:json" json:"config"`
}

type Contract struct {
	Model
	ApplicationId uuid.UUID        `sql:"not null;type:uuid" json:"-"`
	NetworkId     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	TransactionId *uuid.UUID       `sql:"type:uuid" json:"transaction_id"` // id of the transaction which created the contract (or null)
	Name          *string          `sql:"not null" json:"name"`
	Address       *string          `sql:"not null" json:"address"` // network-specific token contract address
	Params        *json.RawMessage `sql:"type:json" json:"params"`
}

type Token struct {
	Model
	ApplicationId  uuid.UUID  `sql:"not null;type:uuid" json:"-"`
	NetworkId      uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	ContractId     *uuid.UUID `sql:"type:uuid" json:"contract_id"`
	SaleContractId *uuid.UUID `sql:"type:uuid" json:"sale_contract_id"`
	Name           *string    `sql:"not null" json:"name"`
	Symbol         *string    `sql:"not null" json:"symbol"`
	Decimals       uint64     `sql:"not null" json:"decimals"`
	Address        *string    `sql:"not null" json:"address"` // network-specific token contract address
	SaleAddress    *string    `json:"sale_address"`           // non-null if token sale contract is specified
}

type Transaction struct {
	Model
	ApplicationId uuid.UUID        `sql:"not null;type:uuid" json:"-"`
	NetworkId     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	WalletId      uuid.UUID        `sql:"not null;type:uuid" json:"wallet_id"`
	To            *string          `json:"to"`
	Value         uint64           `sql:"not null;default:0" json:"value"`
	Data          *string          `json:"data"`
	Hash          *string          `sql:"not null" json:"hash"`
	Params        *json.RawMessage `sql:"-" json:"params"`
}

type Wallet struct {
	Model
	ApplicationId uuid.UUID `sql:"not null;type:uuid" json:"-"`
	NetworkId     uuid.UUID `sql:"not null;type:uuid" json:"network_id"`
	Address       string    `sql:"not null" json:"address"`
	PrivateKey    *string   `sql:"not null;type:bytea" json:"-"`
}

// Network

func (n *Network) ParseConfig() map[string]interface{} {
	config := map[string]interface{}{}
	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)
		if err != nil {
			Log.Warningf("Failed to unmarshal network config; %s", err.Error())
			return nil
		}
	}
	return config
}

// Contract

func (c *Contract) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if c.Params != nil {
		err := json.Unmarshal(*c.Params, &params)
		if err != nil {
			Log.Warningf("Failed to unmarshal contract params; %s", err.Error())
			return nil
		}
	}
	return params
}

func (c *Contract) Create() bool {
	db := DatabaseConnection()

	if !c.Validate() {
		return false
	}

	if db.NewRecord(c) {
		result := db.Create(&c)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				c.Errors = append(c.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(c) {
			return rowsAffected > 0
		}
	}
	return false
}

func (c *Contract) GetTransaction() (*Transaction, error) {
	var tx = &Transaction{}
	db := DatabaseConnection()
	db.Model(c).Related(&tx)
	if tx == nil {
		return nil, fmt.Errorf("Failed to retrieve tx for contract: %s", c.Id)
	}
	return tx, nil
}

func (c *Contract) Validate() bool {
	db := DatabaseConnection()
	var transaction = &Transaction{}
	db.Model(c).Related(&transaction)
	c.Errors = make([]*Error, 0)
	if c.NetworkId == uuid.Nil {
		c.Errors = append(c.Errors, &Error{
			Message: stringOrNil("Unable to associate contract with unspecified network"),
		})
	} else if c.NetworkId != transaction.NetworkId {
		c.Errors = append(c.Errors, &Error{
			Message: stringOrNil("Contract network did not match transaction network"),
		})
	}
	return len(c.Errors) == 0
}

// Token

func (t *Token) Create() bool {
	db := DatabaseConnection()

	if !t.Validate() {
		return false
	}

	if db.NewRecord(t) {
		result := db.Create(&t)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errors = append(t.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(t) {
			return rowsAffected > 0
		}
	}
	return false
}

func (t *Token) Validate() bool {
	db := DatabaseConnection()
	var contract = &Contract{}
	if t.NetworkId != uuid.Nil {
		db.Model(t).Related(&contract)
	}
	t.Errors = make([]*Error, 0)
	if t.NetworkId == uuid.Nil {
		t.Errors = append(t.Errors, &Error{
			Message: stringOrNil("Unable to deploy token contract using unspecified network"),
		})
	} else {
		if contract != nil {
			if t.NetworkId != contract.NetworkId {
				t.Errors = append(t.Errors, &Error{
					Message: stringOrNil("Token network did not match token contract network"),
				})
			}
			if t.Address == nil {
				t.Address = contract.Address
			} else if t.Address != nil && *t.Address != *contract.Address {
				t.Errors = append(t.Errors, &Error{
					Message: stringOrNil("Token contract address did not match referenced contract address"),
				})
			}
		}
		// if t.SaleContractId != nil {
		// 	if t.NetworkId != saleContract.NetworkId {
		// 		t.Errors = append(t.Errors, &Error{
		// 			Message: stringOrNil("Token network did not match token sale contract network"),
		// 		})
		// 	}
		// 	if t.SaleAddress == nil {
		// 		t.SaleAddress = saleContract.Address
		// 	} else if t.SaleAddress != nil && *t.SaleAddress != *saleContract.Address {
		// 		t.Errors = append(t.Errors, &Error{
		// 			Message: stringOrNil("Token sale address did not match referenced token sale contract address"),
		// 		})
		// 	}
		// }
	}
	return len(t.Errors) == 0
}

func (t *Token) GetContract() (*Contract, error) {
	db := DatabaseConnection()
	var contract = &Contract{}
	db.Model(t).Related(&contract)
	if contract == nil {
		return nil, fmt.Errorf("Failed to retrieve token contract for token: %s", t.Id)
	}
	return contract, nil
}

func (t *Token) readEthereumContractAbi() (*ethabi.ABI, error) {
	contract, err := t.GetContract()
	if err != nil {
		return nil, err
	}
	tx, err := contract.GetTransaction()
	if err != nil {
		return nil, err
	}
	return tx.readEthereumContractAbi()
}

// Transaction

func (t *Transaction) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if t.Params != nil {
		err := json.Unmarshal(*t.Params, &params)
		if err != nil {
			Log.Warningf("Failed to unmarshal transaction params; %s", err.Error())
			return nil
		}
	}
	return params
}

func (t *Transaction) asEthereumCallMsg(gasPrice, gasLimit uint64) ethereum.CallMsg {
	db := DatabaseConnection()
	var wallet = &Wallet{}
	db.Model(t).Related(&wallet)
	var to *common.Address
	var data []byte
	if t.To != nil {
		addr := common.HexToAddress(*t.To)
		to = &addr
	}
	if t.Data != nil {
		data = common.FromHex(*t.Data)
	}
	return ethereum.CallMsg{
		From:     common.HexToAddress(wallet.Address),
		To:       to,
		Gas:      gasLimit,
		GasPrice: big.NewInt(int64(gasPrice)),
		Value:    big.NewInt(int64(t.Value)),
		Data:     data,
	}
}

func (t *Transaction) readEthereumContractAbi() (*ethabi.ABI, error) {
	var abi *ethabi.ABI
	params := t.ParseParams()
	if contractAbi, ok := params["abi"]; ok {
		abistr, err := json.Marshal(contractAbi)
		if err != nil {
			Log.Warningf("Failed to marshal abi to json...  %s", err.Error())
		}
		abival, err := ethabi.JSON(strings.NewReader(string(abistr)))
		if err != nil {
			return nil, err
		}
		abi = &abival
	} else {
		return nil, fmt.Errorf("Failed to read abi from params for tx: %s", t.Id)
	}
	return abi, nil
}

func (t *Transaction) signEthereumTx(network *Network, wallet *Wallet, cfg *ethparams.ChainConfig) (*types.Transaction, error) {
	client := JsonRpcClient(network)
	syncProgress, err := client.SyncProgress(context.TODO())
	if err == nil {
		hdr, err := client.HeaderByNumber(context.TODO(), nil)
		if err != nil {
			return nil, err
		}
		nonce, err := client.PendingNonceAt(context.TODO(), common.HexToAddress(wallet.Address))
		if err != nil {
			return nil, err
		}
		gasPrice, _ := client.SuggestGasPrice(context.TODO())
		var data []byte
		if t.Data != nil {
			data = common.FromHex(*t.Data)
		}
		var tx *types.Transaction
		if t.To != nil {
			addr := common.HexToAddress(*t.To)
			gasLimit := big.NewInt(DefaultEthereumGasLimit).Uint64()
			tx = types.NewTransaction(nonce, addr, big.NewInt(int64(t.Value)), gasLimit, gasPrice, data)
		} else {
			Log.Debugf("Attempting to deploy %s contract via tx; estimating total gas requirements", *network.Name)
			callMsg := t.asEthereumCallMsg(gasPrice.Uint64(), 0)
			gasLimit, err := client.EstimateGas(context.TODO(), callMsg)
			if err != nil {
				Log.Warningf("Failed to estimate gas for %s contract deployment tx; %s", *network.Name, err.Error())
				return nil, err
			}
			Log.Debugf("Estimated %d total gas required for %s contract deployment tx with %d-byte data payload", gasLimit, *network.Name, len(data))
			tx = types.NewContractCreation(nonce, big.NewInt(int64(t.Value)), gasLimit, gasPrice, data)
		}
		signer := types.MakeSigner(cfg, hdr.Number)
		hash := signer.Hash(tx).Bytes()
		sig, err := wallet.SignTx(hash)
		if err == nil {
			signedTx, _ := tx.WithSignature(signer, sig)
			t.Hash = stringOrNil(fmt.Sprintf("%x", signedTx.Hash()))
			Log.Debugf("Signed %s tx for broadcast via JSON-RPC: %s", *network.Name, signedTx.String())
			return signedTx, nil
		}
		return nil, err
	} else if syncProgress == nil {
		Log.Debugf("%s JSON-RPC is in sync with the network", *network.Name)
	}
	return nil, err
}

func (t *Transaction) Create() bool {
	if !t.Validate() {
		return false
	}

	db := DatabaseConnection()
	var network = &Network{}
	var wallet = &Wallet{}
	if t.NetworkId != uuid.Nil {
		db.Model(t).Related(&network)
		db.Model(t).Related(&wallet)
	}

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		client, err := DialJsonRpc(network)
		if err != nil {
			Log.Warningf("Failed to dial %s JSON-RPC host; %s", *network.Name, err.Error())
			t.Errors = append(t.Errors, &Error{
				Message: stringOrNil(err.Error()),
			})
		} else {
			cfg := GetChainConfig(network)
			tx, err := t.signEthereumTx(network, wallet, cfg)
			if err == nil {
				Log.Debugf("Transmitting signed %s tx to JSON-RPC host", *network.Name)
				err := client.SendTransaction(context.TODO(), tx)
				if err != nil {
					Log.Warningf("Failed to transmit signed %s tx to JSON-RPC host; %s", *network.Name, err.Error())
					t.Errors = append(t.Errors, &Error{
						Message: stringOrNil(err.Error()),
					})
				}
			} else {
				Log.Warningf("Failed to sign %s tx using wallet: %s; %s", *network.Name, wallet.Id, err.Error())
				t.Errors = append(t.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
	} else {
		Log.Warningf("Unable to generate tx to sign for unsupported network: %s", *network.Name)
	}

	if len(t.Errors) > 0 {
		return false
	}

	if db.NewRecord(t) {
		result := db.Create(&t)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errors = append(t.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(t) {
			if t.To == nil && rowsAffected > 0 {
				if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
					var err error
					var receipt *types.Receipt
					client, err := DialJsonRpc(network)
					gasPrice, _ := client.SuggestGasPrice(context.TODO())
					txHash := fmt.Sprintf("0x%s", *t.Hash)
					Log.Debugf("%s contract created by broadcast tx: %s", *network.Name, txHash)
					err = ethereum.NotFound
					for receipt == nil && err == ethereum.NotFound {
						Log.Debugf("Retrieving tx receipt for %s contract creation tx: %s", *network.Name, txHash)
						receipt, err = client.TransactionReceipt(context.TODO(), common.HexToHash(txHash))
						if err != nil && err == ethereum.NotFound {
							Log.Warningf("%s contract created by broadcast tx: %s; address must be retrieved from tx receipt", *network.Name, txHash)
						} else {
							Log.Debugf("Retrieved tx receipt for %s contract creation tx: %s; deployed contract address: %s", *network.Name, txHash, receipt.ContractAddress.Hex())
							params := t.ParseParams()
							contractName := fmt.Sprintf("Contract %s", *stringOrNil(receipt.ContractAddress.Hex()))
							if name, ok := params["name"].(string); ok {
								contractName = name
							}
							contract := &Contract{
								NetworkId:     t.NetworkId,
								TransactionId: &t.Id,
								Name:          stringOrNil(contractName),
								Address:       stringOrNil(receipt.ContractAddress.Hex()),
								Params:        t.Params,
							}
							if contract.Create() {
								Log.Debugf("Created contract %s for %s contract creation tx: %s", contract.Id, *network.Name, txHash)

								if contractAbi, ok := params["abi"]; ok {
									abistr, err := json.Marshal(contractAbi)
									if err != nil {
										Log.Warningf("failed to marshal abi to json...  %s", err.Error())
									}
									abi, err := ethabi.JSON(strings.NewReader(string(abistr)))
									if err == nil {
										msg := ethereum.CallMsg{
											From:     common.HexToAddress(wallet.Address),
											To:       &receipt.ContractAddress,
											Gas:      0,
											GasPrice: gasPrice,
											Value:    nil,
											Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("name()"))),
										}

										result, _ := client.CallContract(context.TODO(), msg, nil)
										var name string
										err = abi.Methods["name"].Outputs.Unpack(&name, result)
										if err != nil {
											Log.Warningf("Failed to unpack token name; %s", err.Error())
										}

										msg = ethereum.CallMsg{
											From:     common.HexToAddress(wallet.Address),
											To:       &receipt.ContractAddress,
											Gas:      0,
											GasPrice: gasPrice,
											Value:    nil,
											Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("decimals()"))),
										}
										result, _ = client.CallContract(context.TODO(), msg, nil)
										var decimals *big.Int
										abi.Methods["decimals"].Outputs.Unpack(&decimals, result)

										msg = ethereum.CallMsg{
											From:     common.HexToAddress(wallet.Address),
											To:       &receipt.ContractAddress,
											Gas:      0,
											GasPrice: gasPrice,
											Value:    nil,
											Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("symbol()"))),
										}
										result, _ = client.CallContract(context.TODO(), msg, nil)
										var symbol string
										abi.Methods["symbol"].Outputs.Unpack(&symbol, result)

										Log.Debugf("Resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)
										token := &Token{
											NetworkId:  contract.NetworkId,
											ContractId: &contract.Id,
											Name:       stringOrNil(name),
											Symbol:     stringOrNil(symbol),
											Decimals:   decimals.Uint64(),
											Address:    stringOrNil(receipt.ContractAddress.Hex()),
										}
										if token.Create() {
											Log.Debugf("Created token %s for associated %s contract creation tx: %s", token.Id, *network.Name, txHash)
										} else {
											Log.Warningf("Failed to create token for associated %s contract creation tx %s; %d errs: %s", *network.Name, txHash, len(token.Errors), *stringOrNil(*token.Errors[0].Message))
										}
									} else {
										Log.Warningf("Failed to parse JSON ABI for %s contract; %s", *network.Name, err.Error())
									}
								}
							} else {
								Log.Warningf("Failed to create contract for %s contract creation tx %s", *network.Name, txHash)
							}
						}
					}
				}
			}
			return rowsAffected > 0
		}
	}
	return false
}

func (t *Transaction) Validate() bool {
	db := DatabaseConnection()
	var wallet = &Wallet{}
	db.Model(t).Related(&wallet)
	t.Errors = make([]*Error, 0)
	if t.NetworkId == uuid.Nil {
		t.Errors = append(t.Errors, &Error{
			Message: stringOrNil("Unable to sign tx using unspecified network"),
		})
	} else if t.NetworkId != wallet.NetworkId {
		t.Errors = append(t.Errors, &Error{
			Message: stringOrNil("Transaction network did not match wallet network"),
		})
	}
	return len(t.Errors) == 0
}

// Wallet

func (w *Wallet) generate(db *gorm.DB, gpgPublicKey string) {
	var network = &Network{}
	db.Model(w).Related(&network)

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		privateKey, err := ethcrypto.GenerateKey()
		if err == nil {
			w.Address = ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()
			encodedPrivateKey := hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
			db.Raw("SELECT pgp_pub_encrypt(?, dearmor(?)) as private_key", encodedPrivateKey, gpgPublicKey).Scan(&w)
			Log.Debugf("Generated Ethereum address: %s", w.Address)
		}
	} else {
		Log.Warningf("Unable to generate private key for wallet using unsupported network: %s", *network.Name)
	}
}

func (w *Wallet) ECDSAPrivateKey(gpgPrivateKey, gpgEncryptionKey string) (*ecdsa.PrivateKey, error) {
	results := make([]byte, 1)
	db := DatabaseConnection()
	rows, err := db.Raw("SELECT pgp_pub_decrypt(?, dearmor(?), ?) as private_key", w.PrivateKey, gpgPrivateKey, gpgEncryptionKey).Rows()
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&results)
		privateKeyBytes, err := hex.DecodeString(string(results))
		if err != nil {
			Log.Warningf("Failed to decode ecdsa private key from encrypted storage; %s", err.Error())
			return nil, err
		}
		return ethcrypto.ToECDSA(privateKeyBytes)
	}
	return nil, errors.New("Failed to decode ecdsa private key from encrypted storage")
}

func (w *Wallet) SignTx(msg []byte) ([]byte, error) {
	db := DatabaseConnection()

	var network = &Network{}
	db.Model(w).Related(&network)

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		privateKey, err := w.ECDSAPrivateKey(GpgPrivateKey, WalletEncryptionKey)
		if err != nil {
			Log.Warningf("Failed to sign tx using %s wallet: %s", *network.Name, w.Id)
			return nil, err
		}

		Log.Debugf("Signing tx using %s wallet: %s", *network.Name, w.Id)
		sig, err := ethcrypto.Sign(msg, privateKey)
		if err != nil {
			Log.Warningf("Failed to sign tx using %s wallet: %s; %s", *network.Name, w.Id, err.Error())
			return nil, err
		}
		return sig, nil
	}

	err := fmt.Errorf("Unable to sign tx using unsupported network: %s", *network.Name)
	Log.Warningf(err.Error())
	return nil, err
}

func (w *Wallet) Create() bool {
	db := DatabaseConnection()

	w.generate(db, GpgPublicKey)
	if !w.Validate() {
		return false
	}

	if db.NewRecord(w) {
		result := db.Create(&w)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				w.Errors = append(w.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(w) {
			return rowsAffected > 0
		}
	}
	return false
}

func (w *Wallet) Validate() bool {
	w.Errors = make([]*Error, 0)
	_, err := w.ECDSAPrivateKey(GpgPrivateKey, WalletEncryptionKey)
	if err != nil {
		msg := err.Error()
		w.Errors = append(w.Errors, &Error{
			Message: &msg,
		})
	}
	return len(w.Errors) == 0
}

func (w *Wallet) TokenBalance(tokenId string) (uint64, error) {
	balance := uint64(0)
	db := DatabaseConnection()
	var network = &Network{}
	var token = &Token{}
	db.Model(w).Related(&network)
	db.Where("id = ?", tokenId).Find(&token)
	if token == nil {
		return 0, fmt.Errorf("Unable to read token balance for invalid token: %s", tokenId)
	}
	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		abi, err := token.readEthereumContractAbi()
		if err != nil {
			return 0, err
		}
		client, err := DialJsonRpc(network)
		gasPrice, _ := client.SuggestGasPrice(context.TODO())
		to := common.HexToAddress(*token.Address)
		msg := ethereum.CallMsg{
			From:     common.HexToAddress(w.Address),
			To:       &to,
			Gas:      0,
			GasPrice: gasPrice,
			Value:    nil,
			Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("balanceOf(address)"))),
		}
		result, _ := client.CallContract(context.TODO(), msg, nil)
		var out *big.Int
		abi.Methods["balanceOf"].Outputs.Unpack(&out, result)
		if out != nil {
			balance = out.Uint64()
			Log.Debugf("Read %s %s token balance (%v) from token contract address: %s", *network.Name, token.Symbol, balance, token.Address)
		}
	} else {
		Log.Warningf("Unable to read token balance for unsupported network: %s", *network.Name)
	}
	return balance, nil
}

func (w *Wallet) TxCount() (count *uint64) {
	db := DatabaseConnection()
	db.Model(&Transaction{}).Where("wallet_id = ?", w.Id).Count(&count)
	return count
}
