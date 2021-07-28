// +build integration nchain transfer

package integration

import (
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	nchain "github.com/provideplatform/provide-go/api/nchain"
)

func TestTransferHDWalletKovanOrg(t *testing.T) {

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	testcaseOrg := Organization{
		"org" + testId.String(),
		"orgdesc " + testId.String(),
	}

	org, err := orgFactory(*userToken, testcaseOrg.name, testcaseOrg.description)
	if err != nil {
		t.Errorf("error setting up organization. Error: %s", err.Error())
		return
	}

	orgToken, err := orgTokenFactory(*userToken, org.ID)
	if err != nil {
		t.Errorf("error getting org token. Error: %s", err.Error())
		return
	}

	wallet, err := nchain.CreateWallet(*orgToken.Token, map[string]interface{}{
		"mnemonic": "traffic charge swing glimpse will citizen push mutual embrace volcano siege identify gossip battle casual exit enrich unlock muscle vast female initial please day",
	})
	if err != nil {
		t.Errorf("error creating wallet: %s", err.Error())
		return
	}

	// this path produces the ETH address 0x6af845bae76f5cc16bc93f86b83e8928c3dfda19
	path := `m/44'/60'/2'/0/0`

	//params := map[string]interface{}{}
	var txRef uuid.UUID

	txRef, err = uuid.NewV4()
	if err != nil {
		t.Errorf("error creating unique tx ref. Error: %s", err.Error())
		return
	}
	// parameter := fmt.Sprintf(`{"hd_derivation_path": "%s", "value":1000, "wallet_id":"%s", "ref": "%s", "gas_price": 6000000000}`, path, wallet.ID, txRef)
	// t.Logf("parameter is: %s", parameter)
	// json.Unmarshal([]byte(parameter), &params)

	// execute the contract method
	//execResponse, err := nchain.CreateTransaction(*orgToken.Token, params)

	execResponse, err := nchain.CreateTransaction(*orgToken.Token, map[string]interface{}{
		"network_id":      kovanNetworkID,
		"organization_id": org.ID.String(),
		"wallet_id":       wallet.ID,
		"value":           1000,
		"params": map[string]interface{}{
			"wallet_id":          wallet.ID.String(),
			"hd_derivation_path": path,
			"ref":                txRef.String(),
		},
	})
	if err != nil {
		t.Errorf("error executing transaction. Error: %s", err.Error())
		return
	}

	// wait for the transaction to be mined (get a tx hash)
	started := time.Now().Unix()
	for {
		if time.Now().Unix()-started >= transactionTimeout {
			t.Error("timed out awaiting transaction hash")
			return
		}

		tx, err := nchain.GetTransactionDetails(*orgToken.Token, *execResponse.Ref, map[string]interface{}{})
		//this is populated by nchain consumer, so it can take a moment to appear, so we won't quit right away on a 404
		if err != nil {
			t.Logf("tx not yet mined...")
		}

		if err == nil {
			if tx.Block != nil && *tx.Hash != "0x" {
				t.Logf("tx resolved; tx id: %s; hash: %s; block: %d", tx.ID.String(), *tx.Hash, *tx.Block)
				break
			}
			t.Logf("resolving transaction...")
		}
		time.Sleep(transactionSleepTime * time.Second)
	}
}
