package main

import (
	"github.com/MinterTeam/minter-go-node/tests/acceptance/api"
	"github.com/MinterTeam/minter-go-node/tests/acceptance/helpers"
	"log"
	"os"
	"strings"
	"time"
)

var (
	tests = []func() error{
		api.TestApiStatus,
		api.TestApiBlock,
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

		if len(os.Args) > 1 {
			if !strings.Contains(strings.ToLower(testName), strings.ToLower(os.Args[1])) {
				continue
			}
		}

		logger.Printf("Running \"%s\"... \n", testName)

		start := time.Now()
		err := test()

		if err != nil {
			helpers.StopMinter()
			logger.Fatalf("Failed test \"%s\"\nReason: %s", testName, err)
		}

		elapsed := time.Since(start)

		logger.Printf("Completed \"%s\" in %s \n", testName, elapsed)
	}

	logger.Printf("Completed all tests\n")
}
