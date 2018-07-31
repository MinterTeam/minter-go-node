package main

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/api"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/genesis"
	"github.com/MinterTeam/minter-go-node/gui"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/tendermint/tendermint/libs/common"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	"os"
)

func main() {

	err := common.EnsureDir(utils.GetMinterHome()+"/config", 0777)

	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	if *utils.ResetPrivateValidator {
		resetFilePV(config.GetConfig().PrivValidatorFile())
	}

	app := minter.NewMinterBlockchain()
	node := startTendermintNode(app)

	app.RunRPC(node)

	if !*utils.DisableApi {
		go api.RunApi(app, node)
		go gui.Run(":3000")
	}

	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		node.Stop()
		app.Stop()
	})
}

func startTendermintNode(app *minter.Blockchain) *tmNode.Node {

	cfg := config.GetConfig()

	node, err := tmNode.NewNode(
		cfg,
		privval.LoadOrGenFilePV(cfg.PrivValidatorFile()),
		proxy.NewLocalClientCreator(app),
		genesis.GetTestnetGenesis,
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

func resetFilePV(privValFile string) {
	if _, err := os.Stat(privValFile); err == nil {
		pv := privval.LoadFilePV(privValFile)
		pv.Reset()
		log.Info("Reset private validator file to genesis state", "file", privValFile)
	} else {
		pv := privval.GenFilePV(privValFile)
		pv.Save()
		log.Info("Generated private validator file", "file", privValFile)
	}
}
