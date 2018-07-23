package api

import (
	"encoding/json"
	"github.com/tendermint/tendermint/rpc/core/types"
	"io/ioutil"
	"net/http"
)

func TestApiStatus() error {
	result, err := http.Get("http://localhost:8841/api/status")

	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(result.Body)

	if err != nil {
		return err
	}

	var status struct {
		Code   int `json:"code"`
		Result struct {
			LatestBlockHash   string `json:"latest_block_hash"`
			LatestAppHash     string `json:"latest_app_hash"`
			LatestBlockHeight int    `json:"latest_block_height"`
			LatestBlockTime   string `json:"latest_block_time"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &status)

	return err
}

func TestApiBlock() error {
	result, err := http.Get("http://localhost:8841/api/block/1")

	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(result.Body)

	if err != nil {
		return err
	}

	var blockResult struct {
		Code   int                    `json:"code"`
		Result core_types.ResultBlock `json:"result"`
	}

	err = json.Unmarshal(data, &blockResult)

	return err
}
