package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

// # TYPE chain_head_safe gauge
// chain_head_safe 0

//# TYPE chain_inserts_count counter
//chain_inserts_count 0

// curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' http://localhost:8545
// {"jsonrpc":"2.0","id":1,"result":{"currentBlock":"0x68528e","healedBytecodeBytes":"0x0","healedBytecodes":"0x0","healedTrienodeBytes":"0x0","healedTrienodes":"0x0","healingBytecode":"0x0","healingTrienodes":"0x0","highestBlock":"0x84d8af","startingBlock":"0x6829af","syncedAccountBytes":"0x10ca6e935","syncedAccounts":"0x10ddf94","syncedBytecodeBytes":"0x1e89ff879","syncedBytecodes":"0x104665","syncedStorage":"0xd18ac93","syncedStorageBytes":"0xb151286b7"}}
type gethResponse struct {
	Result gethInnerResponse `json:result"`
}
type gethInnerResponse struct {
	CurrentBlock string `json:"currentBlock,omitempty"`
	HighestBlock string `json:"highestBlock,omitempty"`
}

// {"data":{"head_slot":"5258399","sync_distance":"664","is_syncing":true,"is_optimistic":true,"el_offline":true}}
type prysmResponse struct {
	Data prysmInnerResponse `json:"data"`
}
type prysmInnerResponse struct {
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
}

//const GETH_REQUEST = `{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}`
//const PRYSM_REQUEST = `http://localhost:3500/eth/v1/node/syncing"`

func fetchGethStatus() (bool, error) {
	buf := bytes.NewBuffer([]byte(`{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}`))
	resp, err := http.Post("http://geth:8545", "application/json", buf)
	if err != nil {
		return false, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, nil
	}

	var out gethResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return false, err
	}

	highestBlock, err := strconv.ParseInt(out.Result.HighestBlock, 0, 64)
	if err != nil {
		return false, err
	}

	if highestBlock > 0 && out.Result.CurrentBlock == out.Result.HighestBlock {
		return true, nil
	}
	return false, nil
}

func fetchPrysmStatus() (bool, error) {
	resp, err := http.Get("http://prysma:3500/eth/v1/node/syncing")
	if err != nil {
		return false, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var out prysmResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return false, err
	}

	distance, err := strconv.ParseInt(out.Data.SyncDistance, 10, 64)
	if err != nil {
		return false, err
	}

	if out.Data.IsSyncing == false && distance < 5 {
		return true, nil
	}
	return false, nil
}

// return func(w http.ResponseWriter, r *http.Request) {
func handleMetrics(w http.ResponseWriter, r *http.Request) {

	gethIsSynced, err := fetchGethStatus()
	if err != nil {
		log.Printf("error fetching geth status: %v", err)
	}
	prysmIsSynced, err := fetchPrysmStatus()
	if err != nil {
		log.Printf("error fetching prysm status: %v", err)
	}

	// # TYPE chain_head_safe gauge
	// chain_head_safe 0

	//# TYPE chain_inserts_count counter
	//chain_inserts_count 0
	body := `#TYPE takehome_geth_synced gauge
takehome_geth_synced %d

#TYPE takehome_prysm_synced gauge
takehome_prysm_synced %d`

	var gv int = 0
	var pv int = 0
	if gethIsSynced {
		gv = 1
	}
	if prysmIsSynced {
		pv = 1
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(body, gv, pv)))
}

func main() {

	http.HandleFunc("/metrics", handleMetrics)
	http.ListenAndServe(":8085", nil)
	for {
		time.Sleep(3 * time.Second)
		prysm, err := fetchPrysmStatus()
		if err != nil {
			log.Fatalf("fetching prysm node status: %v", err)
		}
		fmt.Printf("%+v\n", prysm)
		geth, err := fetchGethStatus()
		if err != nil {
			log.Fatalf("fetching geth node status: %v", err)
		}
		fmt.Printf("%+v\n", geth)

		/*
			time.Sleep(3 * time.Second)
			buf := bytes.NewBuffer([]byte(`{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}`))
			resp, err := http.Post("http://localhost:8545", "application/json", buf)
			if err != nil {
				log.Fatalf("getting metrics: %v", err)

			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatalf("reading body: %v", err)
			}

			fmt.Printf("%+v\n", string(body))
		*/
	}
}
