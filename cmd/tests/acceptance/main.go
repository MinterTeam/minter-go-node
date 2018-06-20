package main

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/cmd/tests/acceptance/helpers"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	tests = []func() error{
		testApiStatus,
		testSendTransaction,
	}
	logger = log.New(os.Stdout, "main-test-routine ", log.LstdFlags)
)

func main() {
	logger.Println("Starting Minter Go Node acceptance testing...")

	minterNodeReady := make(chan bool)
	go helpers.RunMinter(minterNodeReady)
	<-minterNodeReady

	for _, test := range tests {
		testName := helpers.GetTestName(test)
		logger.Printf("Running test \"%s\"... \n", testName)

		start := time.Now()
		err := test()

		if err != nil {
			logger.Fatalf("Failed test \"%s\"\nReason: %s", testName, err)
		}

		elapsed := time.Now().Sub(start)

		logger.Printf("Completed \"%s\" in %s \n", testName, elapsed)
	}
}

func testApiStatus() error {
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

	if err != nil {
		return err
	}

	return nil
}

func testSendTransaction() error {
	return nil
}
