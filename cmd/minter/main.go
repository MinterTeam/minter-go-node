package main

import (
	"github.com/MinterTeam/minter-go-node/api"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/tendermint/tendermint/abci/server"
	"github.com/tendermint/tendermint/libs/common"
)

func main() {
	app := minter.NewMinterBlockchain()

	// Start the listener
	srv, err := server.NewServer(*utils.MinterAppAddrFlag, "socket", app)
	if err != nil {
		panic(err)
	}
	srv.SetLogger(log.With("module", "abci-server"))
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
