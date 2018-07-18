package main

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/api"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/tendermint/tendermint/libs/common"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
)

func main() {

	app := minter.NewMinterBlockchain()
	node := startTendermint(app)

	if !*utils.DisableApi {
		go api.RunApi(app, node)
	}

	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		node.Stop()
		app.Stop()
	})
}

func startTendermint(app *minter.Blockchain) *tmNode.Node {

	cfg := config.GetConfig()

	node, err := tmNode.NewNode(
		cfg,
		privval.LoadOrGenFilePV(cfg.PrivValidatorFile()),
		proxy.NewLocalClientCreator(app),
		tmNode.DefaultGenesisDocProviderFunc(cfg),
		tmNode.DefaultDBProvider,
		tmNode.DefaultMetricsProvider,
		log.With("module", "tendermint"),
	)

	if err != nil {
		fmt.Errorf("Failed to create a node: %v", err)
	}

	if err = node.Start(); err != nil {
		fmt.Errorf("Failed to start node: %v", err)
	}

	log.Info("Started node", "nodeInfo", node.Switch().NodeInfo())

	return node
}
