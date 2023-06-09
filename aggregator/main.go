package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func main() {
	http.HandleFunc("/metrics", handleMetrics)
	http.ListenAndServe(":8085", nil)
}

// export takehome_geth_synced and takehom_prysm_synced prometheus metrics
func handleMetrics(w http.ResponseWriter, r *http.Request) {

	gethIsSynced, _, err := fetchGethStatus()
	if err != nil {
		log.Printf("error fetching geth status: %v", err)
	}
	prysmIsSynced, distance, _, err := fetchPrysmStatus()
	if err != nil {
		log.Printf("error fetching prysm status: %v", err)
	}

	body := `#TYPE takehome_geth_synced gauge
takehome_geth_synced %d

#TYPE takehome_prysm_synced gauge
takehome_prysm_synced %d

#TYPE takehome_prysm_distance gauge
takehome_prysm_distance %d
`

	// prometheus only supports numeric stats so 1 for synced 0 for not
	var gv int = 0
	var pv int = 0
	if gethIsSynced {
		gv = 1
	}
	if prysmIsSynced {
		pv = 1
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(body, gv, pv, distance)))
}

func fetchGethStatus() (bool, int64, error) {
	// see if geth says it is syncing
	type gethSyncingResponse struct {
		Result bool `json:result"`
	}
	buf := bytes.NewBuffer([]byte(`{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}`))
	resp, err := http.Post("http://geth:8545", "application/json", buf)
	if err != nil {
		return false, 0, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, 0, err
	}
	var syncResponse gethSyncingResponse
	if err := json.Unmarshal(body, &syncResponse); err != nil {
		return false, 0, nil
	}

	if syncResponse.Result { // if result is true, we are not synced
		return false, 0, nil
	}

	// see if geth reports a non-zero block for its head if it is not syncing
	type gethBlockResponse struct {
		Result string `json:result"`
	}
	buf = bytes.NewBuffer([]byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`))
	resp, err = http.Post("http://geth:8545", "application/json", buf)
	if err != nil {
		return false, 0, err
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, 0, err
	}
	var blockResponse gethBlockResponse
	if err := json.Unmarshal(body, &blockResponse); err != nil {
		return false, 0, nil
	}
	blockNumber, err := strconv.ParseInt(blockResponse.Result, 0, 64)
	if err != nil {
		return false, 0, err
	}

	return blockNumber > 0, blockNumber, nil
}

func fetchPrysmStatus() (bool, int64, int64, error) {
	type prysmInnerResponse struct {
		SyncDistance string `json:"sync_distance"`
		IsSyncing    bool   `json:"is_syncing"`
		HeadSlot     string `json:"head_slot,omitempty"`
	}
	type prysmResponse struct {
		Data prysmInnerResponse `json:"data"`
	}

	resp, err := http.Get("http://prysma:3500/eth/v1/node/syncing")
	if err != nil {
		return false, 0, 0, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, 0, 0, err
	}

	var out prysmResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return false, 0, 0, err
	}

	distance, err := strconv.ParseInt(out.Data.SyncDistance, 10, 64)
	if err != nil {
		return false, 0, 0, err
	}
	head, err := strconv.ParseInt(out.Data.HeadSlot, 10, 64)
	if err != nil {
		return false, 0, 0, err
	}

	targetHeight := head + distance

	if out.Data.IsSyncing == false && distance < 5 {
		return true, distance, targetHeight, nil
	}
	return false, distance, targetHeight, nil
}
