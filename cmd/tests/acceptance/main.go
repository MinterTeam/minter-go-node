package main

import (
	"github.com/MinterTeam/minter-go-node/cmd/tests/acceptance/api"
	"github.com/MinterTeam/minter-go-node/cmd/tests/acceptance/helpers"
	"log"
	"os"
	"time"
)

var (
	tests = []func() error{
		api.TestApiStatus,
	}
	logger = log.New(os.Stdout, "main-test-routine ", log.LstdFlags)
)

func main() {
	logger.Println("Starting Minter Go Node acceptance testing...")

	minterNodeReady := make(chan bool)
	go helpers.RunMinter(minterNodeReady)
	<-minterNodeReady

	defer helpers.StopMinter()

	for _, test := range tests {
		testName := helpers.GetTestName(test)
		logger.Printf("Running test \"%s\"... \n", testName)

		start := time.Now()
		err := test()

		if err != nil {
			helpers.StopMinter()
			logger.Fatalf("Failed test \"%s\"\nReason: %s", testName, err)
		}

		elapsed := time.Now().Sub(start)

		logger.Printf("Completed \"%s\" in %s \n", testName, elapsed)
	}

	logger.Printf("Completed all tests\n")
}
