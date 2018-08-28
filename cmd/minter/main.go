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

var cfg = config.GetConfig()

func main() {

	err := common.EnsureDir(utils.GetMinterHome()+"/config", 0777)

	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	if *utils.ResetPrivateValidator {
		resetFilePV(config.GetConfig().PrivValidatorFile())
		os.Exit(0)
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
	node, err := tmNode.NewNode(
		cfg,
		privval.LoadOrGenFilePV(cfg.PrivValidatorFile()),
		proxy.NewLocalClientCreator(app),
		genesis.GetTestnetGenesis,
		tmNode.DefaultDBProvider,
		tmNode.DefaultMetricsProvider(cfg.Instrumentation),
		log.With("module", "tendermint"),
	)

	if err != nil {
		println(fmt.Errorf("Failed to create a node: %v", err))
		os.Exit(1)
	}

	if err = node.Start(); err != nil {
		println(fmt.Errorf("Failed to start node: %v", err))
		os.Exit(1)
	}

	log.Info("Started node", "nodeInfo", node.Switch().NodeInfo())

	return node
}

func resetFilePV(privValFile string) {
	if _, err := os.Stat(privValFile); err == nil {
		pv := privval.LoadFilePV(privValFile)
		pv.Reset()
		log.Error("Reset private validator file to genesis state", "file", privValFile)
	} else {
		pv := privval.GenFilePV(privValFile)
		pv.Save()
		log.Error("Generated private validator file", "file", privValFile)
	}
}
