package contract

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	stan "github.com/nats-io/stan.go"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/network"
)

const natsLogTransceiverEmitSubject = "nchain.logs.emit"
const natsLogTransceiverEmitMaxInFlight = 1024
const natsLogTransceiverEmitInvocationTimeout = time.Second * 10
const natsLogTransceiverEmitTimeout = int64(time.Second * 30)

const natsNetworkContractCreateInvocationSubject = "nchain.contract.create"
const natsNetworkContractCreateInvocationMaxInFlight = 32
const natsNetworkContractCreateInvocationTimeout = time.Minute * 1

const natsShuttleContractDeployedSubject = "shuttle.contract.deployed"
const natsShuttleCircuitDeployedSubject = "shuttle.circuit.deployed"

type natsLogEventMessage struct {
	Address         *string                `json:"address,omitempty"`
	Block           uint64                 `json:"block,omitempty"`
	BlockHash       *string                `json:"blockhash,omitempty"`
	Timestamp       uint64                 `json:"timestamp,omitempty"`
	TransactionHash *string                `json:"transaction_hash,omitempty"`
	Data            *string                `json:"data,omitempty"`
	Topics          []*string              `json:"topics,omitempty"`
	Type            *string                `json:"type,omitempty"`
	Params          map[string]interface{} `json:"params,omitempty"`
	// Index           *big.Int        // FIXME? add logIndex?

	NetworkID *string `json:"network_id,omitempty"`
}

var (
	cachedNetworks            = map[string]*network.Network{}     // map of network id -> network
	cachedNetworkContracts    = map[string]map[string]*Contract{} // map of network id -> contract address -> contract
	cachedNetworkContractABIs = map[string]map[string]*abi.ABI{}  // map of network id -> contract address -> ABI

	db        *gorm.DB
	mutex     *sync.Mutex
	waitGroup sync.WaitGroup
)

func init() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Contract package consumer configured to skip NATS streaming subscription setup")
		return
	}

	natsutil.EstablishSharedNatsStreamingConnection(nil)
	db = dbconf.DatabaseConnection()
	mutex = &sync.Mutex{}

	createNatsLogTransceiverEmitInvocationSubscriptions(&waitGroup)
	createNatsNetworkContractCreateInvocationSubscriptions(&waitGroup)
}

func cachedContractArtifacts(networkID uuid.UUID, addr, txHash string) (*Contract, *abi.ABI) {
	var cachedContracts map[string]*Contract
	if cachedCntrcts, cachedCntrctsOk := cachedNetworkContracts[networkID.String()]; cachedCntrctsOk {
		cachedContracts = cachedCntrcts
	} else {
		mutex.Lock()
		cachedContracts = map[string]*Contract{}
		// CHECKME - this throws a subscribelocked panic occasionally
		cachedNetworkContracts[networkID.String()] = cachedContracts
		mutex.Unlock()
	}

	var cachedContractABIs map[string]*abi.ABI
	if cachedABIs, cachedABIsOk := cachedNetworkContractABIs[networkID.String()]; cachedABIsOk {
		cachedContractABIs = cachedABIs
	} else {
		mutex.Lock()
		cachedContractABIs = map[string]*abi.ABI{}
		cachedNetworkContractABIs[networkID.String()] = cachedContractABIs
		mutex.Unlock()
	}

	var contract *Contract
	var contractABI *abi.ABI
	var err error

	if cachedContract, cachedContractOk := cachedContracts[addr]; cachedContractOk {
		contract = cachedContract
	} else {
		mutex.Lock()
		defer mutex.Unlock()

		common.Log.Tracef("contract cache miss; attempting to load contract from persistent storage for network: %s; address: %s", networkID, addr)

		out := []string{}
		db.Table("transactions").Select("id").Where("transactions.hash = ?", txHash).Pluck("id", &out)
		if len(out) == 0 {
			common.Log.Tracef("contract lookup failed for address: %s; no tx resolved for hash: %s", addr, txHash)
			return nil, nil
		}
		txID, err := uuid.FromString(out[0])
		if err != nil {
			common.Log.Tracef("contract lookup failed for address: %s; no tx resolved for hash: %s; %s", addr, txHash, err.Error())
			return nil, nil
		}

		contract = FindByTxID(db, txID)
		if contract == nil || contract.ID == uuid.Nil {
			common.Log.Tracef("contract lookup failed for address: %s; no contract resolved for tx hash: %s", addr, txHash)
			return nil, nil
		}

		cachedContracts[addr] = contract
	}

	if cachedABI, cachedABIOk := cachedContractABIs[addr]; cachedABIOk {
		contractABI = cachedABI
	} else {
		common.Log.Tracef("contract ABI cache miss; attempting to cache contract ABI for network: %s; address: %s", networkID, addr)
		contractABI, err = contract.ReadEthereumContractAbi()
		if err != nil {
			common.Log.Warningf("failed to read ethereum contract ABI on contract: %s; %s", contract.ID, err.Error())
			return nil, nil
		}

		cachedContractABIs[addr] = contractABI
	}

	return contract, contractABI
}

func cachedNetwork(networkID uuid.UUID) *network.Network {
	var cachedNetwork *network.Network
	if cachedNtwrk, cachedNtwrkOk := cachedNetworks[networkID.String()]; cachedNtwrkOk {
		cachedNetwork = cachedNtwrk
	} else {
		mutex.Lock()
		defer mutex.Unlock()

		common.Log.Debugf("network cache miss; attempting to load network with id: %s", networkID)

		cachedNetwork = &network.Network{}
		db.Where("id = ?", networkID).Find(&cachedNetwork)
		if cachedNetwork == nil || cachedNetwork.ID == uuid.Nil {
			common.Log.Debugf("network lookup failed; unable to continue log message ingestion for network: %s", networkID)
			return nil
		}

		cachedNetworks[networkID.String()] = cachedNetwork
	}

	return cachedNetwork
}

func consumeEVMLogTransceiverEventMsg(networkUUID uuid.UUID, msg *stan.Msg, evtmsg *natsLogEventMessage) {
	if evtmsg.Topics != nil && len(evtmsg.Topics) > 0 && evtmsg.Data != nil {
		evtmsg.Address = common.StringOrNil(ethcommon.HexToAddress(*evtmsg.Address).Hex())

		eventID := ethcommon.HexToHash(*evtmsg.Topics[0])
		eventIDHex := eventID.Hex()
		common.Log.Tracef("attempting to publish parsed log emission event with id: %s", eventIDHex)

		contract, contractABI := cachedContractArtifacts(networkUUID, *evtmsg.Address, *evtmsg.TransactionHash)
		if contract == nil {
			common.Log.Tracef("no contract resolved for log emission event with id: %s; nacking log event", eventIDHex)
			natsutil.Nack(msg)
			return
		}
		if contractABI == nil {
			common.Log.Tracef("no contract abi resolved for log emission event with id: %s; nacking log event", eventIDHex)
			natsutil.Nack(msg)
			return
		}

		abievt, err := contractABI.EventByID(eventID)
		if err != nil {
			common.Log.Warningf("failed to publish log emission event with id: %s; %s", eventIDHex, err.Error())
			natsutil.Nack(msg)
			return
		}

		mappedValues := map[string]interface{}{}
		err = abievt.Inputs.UnpackIntoMap(mappedValues, hexutil.MustDecode(*evtmsg.Data))
		if err != nil {
			common.Log.Warningf("failed to ingest log event with id: %s; unpacking values failed; %s", eventIDHex, err.Error())
			return
		}

		var subject string
		if sub, subOk := mappedValues["subject"].(string); subOk {
			subject = sub
		}

		evtmsg.Params = mappedValues
		if typ, typeOk := mappedValues["contractType"].(string); typeOk {
			evtmsg.Params["type"] = typ
			delete(evtmsg.Params, "contractType")
		}

		evtmsg.Params["by"] = *evtmsg.Address
		evtmsg.Params["tx_hash"] = *evtmsg.TransactionHash
		evtmsg.Params["network_id"] = networkUUID.String()

		payload, _ := json.Marshal(evtmsg.Params)
		common.Log.Tracef("unpacked emitted log event values with id: %s; emitting %d-byte payload", eventIDHex, len(payload))

		networkQualifiedSubject := contract.networkQualifiedSubject(nil)
		if networkQualifiedSubject != nil {
			err = natsutil.NatsPublish(*networkQualifiedSubject, payload)
			if err != nil {
				common.Log.Warningf("failed to publish %d-byte contract log event with id: %s; subject: %s; %s", len(payload), eventIDHex, *networkQualifiedSubject, err.Error())
				natsutil.AttemptNack(msg, natsLogTransceiverEmitTimeout)
				return
			}
		}

		qualifiedSubject := contract.qualifiedSubject(subject)
		if qualifiedSubject != nil {
			err = natsutil.NatsPublish(*qualifiedSubject, payload)
			if err != nil {
				common.Log.Warningf("failed to publish %d-byte log event with id: %s; %s", len(payload), eventIDHex, err.Error())
				natsutil.AttemptNack(msg, natsLogTransceiverEmitTimeout)
			} else {
				common.Log.Debugf("published %d-byte log event with id: %s; subject: %s", len(payload), eventIDHex, *qualifiedSubject)
				if subject == natsShuttleContractDeployedSubject || subject == natsShuttleCircuitDeployedSubject { // HACK!!!
					natsutil.NatsStreamingPublish(subject, payload)
				}
				msg.Ack()
			}
		} else {
			common.Log.Tracef("dropping %d-byte log emission event on the floor; contract not configured for pub/sub fanout", len(msg.Data))
			natsutil.Nack(msg)
		}
	} else {
		common.Log.Tracef("dropping anonymous %d-byte log emission event on the floor", len(msg.Data))
		natsutil.Nack(msg)
	}
}

func createNatsLogTransceiverEmitInvocationSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsLogTransceiverEmitInvocationTimeout,
			natsLogTransceiverEmitSubject,
			natsLogTransceiverEmitSubject,
			consumeLogTransceiverEmitMsg,
			natsLogTransceiverEmitInvocationTimeout,
			natsLogTransceiverEmitMaxInFlight,
			nil,
		)
	}
}

func createNatsNetworkContractCreateInvocationSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsNetworkContractCreateInvocationTimeout,
			natsNetworkContractCreateInvocationSubject,
			natsNetworkContractCreateInvocationSubject,
			consumeNetworkContractCreateInvocationMsg,
			natsNetworkContractCreateInvocationTimeout,
			natsNetworkContractCreateInvocationMaxInFlight,
			nil,
		)
	}
}

func consumeLogTransceiverEmitMsg(msg *stan.Msg) {
	common.Log.Tracef("consuming NATS log transceiver event emission message: %s", msg)

	evtmsg := &natsLogEventMessage{}
	err := json.Unmarshal(msg.Data, &evtmsg)
	if err != nil {
		common.Log.Warningf("failed to umarshal log transceiver event emission message; %s", err.Error())
		natsutil.Nack(msg)
		return
	}

	var networkID string
	if evtmsg.NetworkID != nil {
		networkID = *evtmsg.NetworkID
	}
	networkUUID, networkUUIDErr := uuid.FromString(networkID)

	if evtmsg.Address == nil {
		common.Log.Warningf("failed to process log transceiver event emission message; no contract address provided")
		natsutil.Nack(msg)
		return
	}
	if networkUUIDErr != nil {
		common.Log.Warningf("failed to process log transceiver event emission message; invalid or no network id provided")
		natsutil.Nack(msg)
		return
	}

	common.Log.Tracef("unmarshaled %d-byte log transceiver event from emitted log event JSON", len(msg.Data))

	// CHECKME - root of the panic thrown occasionally
	network := cachedNetwork(networkUUID)
	if network == nil || network.ID == uuid.Nil {
		common.Log.Warningf("failed to process log transceiver event emission message; network lookup failed for network id: %s", networkID)
		natsutil.Nack(msg)
		return
	}

	if network.IsEthereumNetwork() {
		consumeEVMLogTransceiverEventMsg(networkUUID, msg, evtmsg)
	} else {
		common.Log.Warningf("failed to process log transceiver event emission message; log events not supported for network: %s", networkID)
		natsutil.Nack(msg)
		return
	}
}

func consumeNetworkContractCreateInvocationMsg(msg *stan.Msg) {
	common.Log.Debugf("consuming NATS network contract creation invocation message: %s", msg)

	var params map[string]interface{}
	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("failed to umarshal network contract creation invocation message; %s", err.Error())
		natsutil.Nack(msg)
		return
	}

	addr, addrOk := params["address"].(string)
	networkID, networkIDOk := params["network_id"].(string)
	networkUUID, networkUUIDErr := uuid.FromString(networkID)
	contractName, contractNameOk := params["name"].(string)
	_, abiOk := params["abi"].([]interface{})

	if !addrOk {
		common.Log.Warningf("failed to create network contract; no contract address provided")
		natsutil.Nack(msg)
		return
	}
	if !networkIDOk || networkUUIDErr != nil {
		common.Log.Warningf("failed to create network contract; invalid or no network id provided")
		natsutil.Nack(msg)
		return
	}
	if !contractNameOk {
		common.Log.Warningf("failed to create network contract; no contract name provided")
		natsutil.Nack(msg)
		return
	}
	if !abiOk {
		common.Log.Warningf("failed to create network contract; no ABI provided")
		natsutil.Nack(msg)
		return
	}
	contract := &Contract{
		ApplicationID: nil,
		NetworkID:     networkUUID,
		TransactionID: nil,
		Name:          common.StringOrNil(contractName),
		Address:       common.StringOrNil(addr),
		Params:        nil,
	}
	contract.setParams(params)

	if contract.Create() {
		common.Log.Debugf("network contract creation invocation succeeded; ACKing NATS message for contract: %s", contract.ID)
		msg.Ack()
	} else {
		common.Log.Warningf("failed to persist network contract with address: %s", addr)
		natsutil.Nack(msg)
	}
}
