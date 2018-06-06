package main

import (
		"github.com/tendermint/abci/server"
	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
	"minter/api"
	"minter/core/minter"
		"os"
		)

func main() {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	app := minter.NewMinterBlockchain()

	// Start the listener
	srv, err := server.NewServer("tcp://0.0.0.0:46658", "socket", app)
	if err != nil {
		panic(err)
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if err := srv.Start(); err != nil {
		panic(err)
	}

	go api.RunApi(app)

	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		app.Stop()
		srv.Stop()
	})
}
