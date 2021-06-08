package tx

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kthomas/go-redisutil"
	"github.com/provideapp/nchain/common"
	provide "github.com/provideservices/provide-go/api"
	providecrypto "github.com/provideservices/provide-go/crypto"
)

const currentBroadcastNonce = "nchain.broadcast.nonce"
const proposedBroadcastNonce = "nchain.tx.nonce"

const RedisNonceTTL = time.Second * 5

type BroadcastConfirmation struct {
	signer    *TransactionSigner //HACK remove this
	address   *string
	network   *string
	nonce     *uint64
	Signed    bool
	Broadcast bool
}

// incoming and outgoing channels for transactions
type channelPair struct {
	incoming chan BroadcastConfirmation
	outgoing chan BroadcastConfirmation
}

// dictionary for channels (interface is channelPair struct only)
var channelPairs map[string]interface{}

type ValueDictionary struct {
	channels map[string]channelPair
	lock     sync.RWMutex
}

var dict ValueDictionary

// Set adds a new item to the dictionary
func (d *ValueDictionary) Set(k string, v channelPair) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.channels == nil {
		d.channels = make(map[string]channelPair)
	}
	d.channels[k] = v
}

// Get returns the value associated with the key
func (d *ValueDictionary) Get(k string) channelPair {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.channels[k]
}

// Has returns true if the key exists in the dictionary
func (d *ValueDictionary) Has(k string) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()
	_, ok := d.channels[k]
	return ok
}

func getWithLock(key string) (*string, error) {
	var status *string

	lockErr := redisutil.WithRedlock(key, func() error {
		var err error
		status, err = redisutil.Get(key)
		if err != nil {
			return err
		}
		return nil
	})
	if lockErr != nil {
		return nil, lockErr
	}
	common.Log.Debugf("Got message key %s status %s", key, *status)
	return status, nil
}

func setWithLock(key, status string) error {
	lockErr := redisutil.WithRedlock(key, func() error {
		err := redisutil.Set(key, status, nil)
		if err != nil {
			return err
		}
		return nil
	})
	if lockErr != nil {
		return lockErr
	}
	common.Log.Debugf("Set message key %s to status %s", key, status)
	return nil
}

func (t *Transaction) getSigner(db *gorm.DB) error {
	start := time.Now()

	signer, err := t.signerFactory(db)
	if err != nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		return err
	}

	t.EthSigner = signer
	elapsedGeneratingSigner := time.Since(start)
	common.Log.Debugf("TIMING: Getting signer for tx ref %s took %s", *t.Ref, elapsedGeneratingSigner)
	return nil
}

func (t *Transaction) nonceProvided() bool {

	// check if the transaction already has a nonce provided
	params := t.ParseParams()
	var nonce *uint64
	if nonceFloat, nonceOk := params["nonce"].(float64); nonceOk {
		nonceUint := uint64(nonceFloat)
		nonce = &nonceUint
	}

	// if the tx has a nonce provided, use it
	if nonce != nil {
		t.Nonce = nonce
		common.Log.Debugf("Transaction ref %s has nonce %v provided.", *t.Ref, *nonce)
		return true
	}
	// otherwise, we have no nonce provided
	return false

}

func (t *Transaction) getNonce(db *gorm.DB) (*uint64, *TransactionSigner, error) {
	nonceMutex.Lock()

	defer func() {
		nonceMutex.Unlock()
	}()

	err := t.getSigner(db)
	if err != nil {
		return nil, nil, err
	}

	if t.nonceProvided() {
		return t.Nonce, t.EthSigner, nil
	}

	// then use the signer to get the transaction address pointer
	txAddress := common.StringOrNil(t.EthSigner.Address())
	t.Nonce, err = t.getNextNonce()
	if err != nil {
		common.Log.Debugf("Error getting nonce for Address %s, tx ref %s. Error: %s", *txAddress, *t.Ref, err.Error())
		return nil, nil, err
	}

	return t.Nonce, t.EthSigner, nil
}

// TODO currently using redis
// but need to remove dep on redis for this,
// and move to DB
// to facilitate retries
// TODO name this bettar
func (t *Transaction) getNextNonce() (*uint64, error) {

	nonceStart := time.Now()
	common.Log.Debugf("XXX: Getting nonce for tx ref %s", *t.Ref)

	var nonce *uint64

	network := t.EthSigner.Network.ID.String()
	address := t.EthSigner.Address()

	key := fmt.Sprintf("%s.%s:%s", proposedBroadcastNonce, address, network)

	cachedNonce, _ := redisutil.Get(key)

	if cachedNonce == nil {
		common.Log.Debugf("XXX: No nonce found on redis for address: %s on network %s, tx ref: %s", address, network, *t.Ref)
		// get the nonce from the EVM
		common.Log.Debugf("XXX: dialling evm for tx ref %s", *t.Ref)
		client, err := providecrypto.EVMDialJsonRpc(t.EthSigner.Network.ID.String(), t.EthSigner.Network.RPCURL())
		if err != nil {
			return nil, err
		}
		common.Log.Debugf("XXX: Getting nonce from chain for tx ref %s", *t.Ref)
		// get the last mined nonce, we don't want to rely on the tx pool
		pendingNonce, err := client.NonceAt(context.TODO(), providecrypto.HexToAddress(address), nil)
		if err != nil {
			common.Log.Debugf("XXX: Error getting pending nonce for tx ref %s. Error: %s", *t.Ref, err.Error())
			return nil, err
		}
		common.Log.Debugf("XXX: Pending nonce found for tx Ref %s. Nonce: %v", *t.Ref, pendingNonce)
		// put this in redis
		lockErr := redisutil.WithRedlock(key, func() error {
			ttl := RedisNonceTTL
			err := redisutil.Set(key, pendingNonce, &ttl)
			if err != nil {
				return err
			}
			return nil
		})
		if lockErr != nil {
			return nil, lockErr
		}
		return &pendingNonce, nil
	} else {
		int64nonce, err := strconv.ParseUint(string(*cachedNonce), 10, 64)
		if err != nil {
			common.Log.Debugf("XXX: Error converting cached nonce to int64 for tx ref: %s. Error: %s", *t.Ref, err.Error())
			return nil, err
		} else {
			common.Log.Debugf("XXX: Assigning nonce of %v to tx ref: %s", int64nonce, *t.Ref)
			nonce = &int64nonce
			//experiment. if it's in redis, we'll always increment it by 1
			nonce, err = incrementNonce(key, *t.Ref, *nonce)
			if err != nil {
				common.Log.Debugf("XXX: Error incrementing nonce to %v tx ref: %s. Error: %s", int64nonce+1, *t.Ref, err.Error())
				return nil, err
			}
		}
	}

	// assign the proposed nonce to the tx
	t.Nonce = nonce

	elapsed := time.Since(nonceStart)
	common.Log.Debugf("TIMING: Getting nonce %v for tx ref %s took %s", *t.Nonce, *t.Ref, elapsed)
	return nonce, nil
}

func incrementNonce(key, txRef string, txNonce uint64) (*uint64, error) {

	common.Log.Debugf("XXX: Incrementing redis nonce for tx ref %s", txRef)

	cachedNonce, _ := getWithLock(key)

	if cachedNonce == nil {
		common.Log.Debugf("XXX: No nonce found on redis to increment for key: %s, tx ref: %s", key, txRef)
		updatedNonce := txNonce + 1
		lockErr := redisutil.WithRedlock(key, func() error {
			ttl := RedisNonceTTL
			err := redisutil.Set(key, updatedNonce, &ttl)
			if err != nil {
				return err
			}
			return nil
		})
		if lockErr != nil {
			return nil, lockErr
		}

		return &updatedNonce, nil
		// the evmtxfactory will get the current nonce from the chain
	}

	if cachedNonce != nil {
		common.Log.Debugf("XXX: Nonce of %v found on redis for key: %s, tx ref: %s", *cachedNonce, key, txRef)
		//convert cached Nonce and increment, returning updated nonce for reference
		int64nonce, err := strconv.ParseUint(string(*cachedNonce), 10, 64)
		if err != nil {
			common.Log.Debugf("XXX: Error converting cached nonce to int64 for tx ref: %s. Error: %s", txRef, err.Error())
			return nil, err
		}
		updatedNonce := int64nonce + 1
		common.Log.Debugf("XXX: Incrementing redis nonce for key %s after tx ref %s to %v", key, txRef, updatedNonce)
		lockErr := redisutil.WithRedlock(key, func() error {
			ttl := RedisNonceTTL
			err := redisutil.Set(key, updatedNonce, &ttl)
			if err != nil {
				return err
			}
			return nil
		})
		if lockErr != nil {
			return nil, lockErr
		}
		common.Log.Debugf("XXX: Nonce incremented for key %s after tx ref %s to %v", key, txRef, updatedNonce)
		return &updatedNonce, nil
		// the evmtxfactory will get the current nonce from the chain
	}

	return nil, nil
}
