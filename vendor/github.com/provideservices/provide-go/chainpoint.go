package provide

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	underscore "github.com/ahl5esoft/golang-underscore"
)

// Node represents a chainpoint node
type Node struct {
	PublicURI string `json:"public_uri"`
}

// NodeList array of Node pointers
type NodeList []*Node

var cachedChainpointNodes *NodeList

type submitHashesResponse struct {
	Meta struct {
		SubmittedAt     time.Time `json:"submitted_at"`
		SubmittedTo     string    `json:"submitted_to"`
		ProcessingHints struct {
			Cal time.Time `json:"cal"`
			Btc time.Time `json:"btc"`
		} `json:"processing_hints"`
	} `json:"meta"`
	Hashes []struct {
		HashIDNode string `json:"hash_id_node"`
		Hash       string `json:"hash"`
	} `json:"hashes"`
}

// ProofHandle
type ProofHandle struct {
	Hash       string `json:"hash"`
	HashIDNode string `json:"hashIdNode"`
	URI        string `json:"URI"`
}

// ProofBody
type ProofBody struct {
	Anchors    []interface{} `json:"anchors_complete"`
	HashIDNode string        `json:"hash_id_node"`
	Proof      interface{}   `json:"proof"`
}

// VerifiedProofs as returned by chainpoint
type VerifiedProofs []struct {
	ProofIndex          int       `json:"proof_index"`
	Hash                string    `json:"hash"`
	HashIDNode          string    `json:"hash_id_node"`
	HashSubmittedNodeAt time.Time `json:"hash_submitted_node_at"`
	HashIDCore          string    `json:"hash_id_core"`
	HashSubmittedCoreAt time.Time `json:"hash_submitted_core_at"`
	Anchors             []struct {
		Branch string `json:"branch"`
		Type   string `json:"type"`
		Valid  bool   `json:"valid"`
	} `json:"anchors"`
	Status string `json:"status"`
}

func init() {
	cacheRandomNodes()
}

// cacheRandomNodes added for readability and for future-state impl
// when we detect nodes that have gone offline; it may make sense for
// fault tolerance to create a readlock on cachedChainpointNodes when
// we have had enough failures out of some % of the cached nodes and
// simply call cacheRandomNodes to re-up with the latest known-good.
func cacheRandomNodes() {
	randomNodes, err := GetNodes()
	if err != nil {
		Log.Errorf("Failed to retrieve chainpoint nodes from service discovery")
	} else {
		cachedChainpointNodes = randomNodes
	}
}

// GetNodes retrieves a list of 25 random Chainpoint Nodes.
func GetNodes() (*NodeList, error) {
	var randomNodes *NodeList
	res, err := http.Get("http://a.chainpoint.org/nodes/random")
	if err != nil {
		Log.Errorf("Failed to make HTTP GET request to fetch a list of random Chainpoint Nodes; %s", err.Error())
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Log.Errorf("Failed to read GET request to retrieve random Chainpoint Nodes; %s", err.Error())
		return nil, err
	}
	json.Unmarshal(body, &randomNodes)

	return randomNodes, nil
}

// SubmitHashes will submit a list of hashes to every Chainpoint Node
// provided to it via the function's second argument `nodes` of type `NodeList`.
// Submitting to multiple Chainpoint Nodes to achieve your required level of redundancy is advised.
// Only a single Chainpoint Proof is required to attest the existence of your data.
func SubmitHashes(hashes []byte, nodes *NodeList) ([]ProofHandle, error) {
	nodeResponseQueue := make(chan submitHashesResponse)

	if nodes == nil {
		nodes = cachedChainpointNodes
	}

	var wg sync.WaitGroup
	wg.Add(len(*nodes))

	for _, node := range *nodes {
		go submitHashesToNode(nodeResponseQueue, &wg, *node, hashes)
	}

	go func() {
		wg.Wait()
		close(nodeResponseQueue)
	}()

	var proofHandleSlice []ProofHandle
	for nodeResponse := range nodeResponseQueue {
		nodeURI := nodeResponse.Meta.SubmittedTo

		for i := 0; i < len(nodeResponse.Hashes); i++ {
			proofHandleSlice = append(proofHandleSlice, ProofHandle{URI: nodeURI, HashIDNode: nodeResponse.Hashes[i].HashIDNode, Hash: nodeResponse.Hashes[i].Hash})
		}
	}

	if len(proofHandleSlice) >= 1 {
		return proofHandleSlice, nil
	} else {
		return nil, errors.New("error submitting hashes to Chainpoint Node(s)")
	}
}

// GetProofs function accepts a list of Proof Handles and will retrieve their corresponding Chainpoint Proofs.
// This function includes the subtle optimization of grouping Proof Handles by Chainpoint Node PublicURI's and will issue
// one HTTP GET request with an embedded header value consisting of a comma-delimited string of hashIdNode values dictating which
// Proofs need to be resolved and returned to the requestor.
func GetProofs(proofHandles []ProofHandle) ([]ProofBody, error) {
	var wg sync.WaitGroup
	getProofsResponseQueue := make(chan []ProofBody)

	v := underscore.GroupBy(proofHandles, "URI")

	proofHandlesMap, ok := v.(map[string][]ProofHandle)
	if !ok {
		Log.Warningf("Failed to group ProofHandles by URI")
		return nil, errors.New("Error creating ProofHandles Map")
	}

	for URI, values := range proofHandlesMap {
		wg.Add(1)
		handlesByURI := underscore.Map(values, func(currVal ProofHandle, _ int) string {
			return currVal.HashIDNode
		})
		hashIds, _ := handlesByURI.([]string)

		go getProofsFromNode(getProofsResponseQueue, &wg, URI, hashIds)
	}

	go func() {
		wg.Wait()
		close(getProofsResponseQueue)
	}()

	var proofsSlice []ProofBody
	for proofsFromNode := range getProofsResponseQueue {
		for _, proof := range proofsFromNode {
			proofsSlice = append(proofsSlice, proof)
		}
	}

	if len(proofsSlice) > 1 {
		return proofsSlice, nil
	} else {
		return nil, errors.New("error fetching proofs from chainpoint nodes")
	}
}

// VerifyProofs accepts a list of base64 encode strings that will be submitted to any Chainpoint Node for verification.
// The verification process will perform the required cryptorgraphic operations for both `cal` and `btc` branches to
// verify the integrity of the Chainpoint Proof(s) that have been submitted.
func VerifyProofs(proofs []string) (VerifiedProofs, error) {
	var verifiedProofs VerifiedProofs
	body := map[string][]string{
		"proofs": proofs,
	}
	bodyBytes, _ := json.Marshal(body)

	res, err := http.Post(fmt.Sprintf("%s/verify", (*cachedChainpointNodes)[0].PublicURI), "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		Log.Warningf("Failed to verify Proofs; %s", err.Error())
		return nil, err
	}
	defer res.Body.Close()

	json.NewDecoder(res.Body).Decode(&verifiedProofs)

	return verifiedProofs, nil
}

// getProofsFromNode retrieves a list of Proofs from a particular Chainpoint Node. Multiple HashIDNode
// values are supported and will be transformed into a comma-delimited list that will be included as
// part of a header for the HTTP GET request to a Node's /proofs endpoint.
func getProofsFromNode(queue chan []ProofBody, w *sync.WaitGroup, URI string, hashIds []string) {
	defer w.Done()
	var getProofResponse []ProofBody

	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/proofs", URI), nil)
	req.Header.Set("hashids", strings.Join(hashIds, ","))

	res, err := client.Do(req)
	if err != nil {
		w.Add(-1)
		Log.Warningf("Failed to retrieve Proofs from URI=%s; %s\n", URI, err.Error())
		return
	}
	defer res.Body.Close()

	json.NewDecoder(res.Body).Decode(&getProofResponse)

	queue <- getProofResponse
}

// Will submit a list of hashes to a Chainpoint Node. This function will make a HTTP Post request
// to the /hashes endpoint of the node and on a 200 response, return an array of ProofHandles.
func submitHashesToNode(queue chan submitHashesResponse, w *sync.WaitGroup, URI Node, hashes []byte) {
	defer w.Done()
	var submitHashesResponse submitHashesResponse

	hashesList := make([]string, len(hashes))
	for _, hash := range hashes {
		hashesList = append(hashesList, string(hash))
	}

	body := map[string][]string{
		"hashes": hashesList,
	}
	bodyBytes, _ := json.Marshal(body)

	res, err := http.Post(fmt.Sprintf("%s/hashes", URI.PublicURI), "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		w.Add(-1)
		Log.Warningf("Failed to submit hashes to Chainpoint Node URI=%s; %s\n", URI.PublicURI, err.Error())
		return
	}
	defer res.Body.Close()

	json.NewDecoder(res.Body).Decode(&submitHashesResponse)

	// Mutate the submitHashesResponse to include the URI that received the hashes
	submitHashesResponse.Meta.SubmittedTo = URI.PublicURI

	queue <- submitHashesResponse
}
