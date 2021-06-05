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

func (t *Transaction) getNonce(db *gorm.DB) (*uint64, *TransactionSigner, error) {
	nonceMutex.Lock()

	defer func() {
		nonceMutex.Unlock()
	}()

	start := time.Now()

	signer, err := t.signerFactory(db)
	if err != nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		return nil, nil, err
	}

	elapsedGeneratingSigner := time.Since(start)
	common.Log.Debugf("TIMING: Getting signer for tx ref %s took %s", *t.Ref, elapsedGeneratingSigner)

	// check if the transaction already has a nonce provided
	params := t.ParseParams()
	var nonce *uint64
	if nonceFloat, nonceOk := params["nonce"].(float64); nonceOk {
		nonceUint := uint64(nonceFloat)
		nonce = &nonceUint
	}

	// if the tx has a nonce provided, use it
	if nonce != nil {
		common.Log.Debugf("Transaction ref %s has nonce %v provided.", *t.Ref, *nonce)
		return nonce, signer, nil
	}
	// signer timer
	signerStart := time.Now()

	// then use the signer factory to get the transaction address
	txAddress, _, err := signer.GetSignerDetails()
	if err != nil {
		common.Log.Debugf("error getting signer details for tx ref %s. Error: %s", *t.Ref, err.Error())
		return nil, nil, err
	}
	signerElapsed := time.Since(signerStart)
	common.Log.Debugf("TIMING: Getting signer for tx ref %s took %s", *t.Ref, signerElapsed)

	nonceStart := time.Now()
	// then use the transaction address to get the nonce
	common.Log.Debugf("XXX: Getting nonce for tx ref %s", *t.Ref)
	nonce, err = getNonce(*txAddress, t, signer)
	if err != nil {
		common.Log.Debugf("Error getting nonce for Address %s, tx ref %s. Error: %s", *txAddress, *t.Ref, err.Error())
		return nil, nil, err
	}

	elapsed := time.Since(nonceStart)
	common.Log.Debugf("TIMING: Getting nonce %v for tx ref %s took %s", *nonce, *t.Ref, elapsed)
	return nonce, signer, nil
}

// TODO currently using redis
// but need to remove dep on redis for this,
// and move to DB
// to facilitate retries
func getNonce(txAddress string, tx *Transaction, txs *TransactionSigner) (*uint64, error) {

	var nonce *uint64

	network := txs.Network.ID.String()
	// TODO const this
	key := fmt.Sprintf("nchain.tx.nonce.%s:%s", txAddress, network)

	cachedNonce, _ := redisutil.Get(key)

	if cachedNonce == nil {
		common.Log.Debugf("XXX: No nonce found on redis for address: %s on network %s, tx ref: %s", txAddress, network, *tx.Ref)
		// get the nonce from the EVM
		common.Log.Debugf("XXX: dialling evm for tx ref %s", *tx.Ref)
		client, err := providecrypto.EVMDialJsonRpc(txs.Network.ID.String(), txs.Network.RPCURL())
		if err != nil {
			return nil, err
		}
		common.Log.Debugf("XXX: Getting nonce from chain for tx ref %s", *tx.Ref)
		// get the last mined nonce, we don't want to rely on the tx pool
		pendingNonce, err := client.NonceAt(context.TODO(), providecrypto.HexToAddress(txAddress), nil)
		if err != nil {
			common.Log.Debugf("XXX: Error getting pending nonce for tx ref %s. Error: %s", *tx.Ref, err.Error())
			return nil, err
		}
		common.Log.Debugf("XXX: Pending nonce found for tx Ref %s. Nonce: %v", *tx.Ref, pendingNonce)
		// put this in redis
		lockErr := redisutil.WithRedlock(txAddress, func() error {
			err := redisutil.Set(key, pendingNonce, nil)
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
			common.Log.Debugf("XXX: Error converting cached nonce to int64 for tx ref: %s. Error: %s", *tx.Ref, err.Error())
			return nil, err
		} else {
			common.Log.Debugf("XXX: Assigning nonce of %v to tx ref: %s", int64nonce, *tx.Ref)
			nonce = &int64nonce
			//experiment. if it's in redis, we'll always increment it by 1
			nonce, err = incrementNonce(key, *tx.Ref, *nonce)
			if err != nil {
				common.Log.Debugf("XXX: Error incrementing nonce to %v tx ref: %s. Error: %s", int64nonce+1, *tx.Ref, err.Error())
				return nil, err
			}
		}
	}
	return nonce, nil
}

func incrementNonce(key, txRef string, txNonce uint64) (*uint64, error) {

	common.Log.Debugf("XXX: Incrementing redis nonce for tx ref %s", txRef)

	cachedNonce, _ := getWithLock(key)

	if cachedNonce == nil {
		common.Log.Debugf("XXX: No nonce found on redis to increment for key: %s, tx ref: %s", key, txRef)
		updatedNonce := txNonce + 1
		lockErr := redisutil.WithRedlock(key, func() error {
			err := redisutil.Set(key, updatedNonce, nil)
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
			err := redisutil.Set(key, updatedNonce, nil)
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
