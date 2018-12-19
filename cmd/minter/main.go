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
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	rpc "github.com/tendermint/tendermint/rpc/client"
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

	client := rpc.NewLocal(node)
	status, _ := client.Status()
	if status.NodeInfo.Network != genesis.Network {
		log.Error("Different networks")
		os.Exit(1)
	}

	app.SetRpcClient(client)

	if !cfg.ValidatorMode {
		go api.RunApi(app, client)
		go gui.Run(cfg.GUIListenAddress)
	}

	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		node.Stop()
		app.Stop()
	})
}

func startTendermintNode(app *minter.Blockchain) *tmNode.Node {
	cfg := config.GetTmConfig()
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())

	if err != nil {
		panic(err)
	}

	node, err := tmNode.NewNode(
		cfg,
		privval.LoadOrGenFilePV(cfg.PrivValidatorFile()),
		nodeKey,
		proxy.NewLocalClientCreator(app),
		genesis.GetTestnetGenesis,
		tmNode.DefaultDBProvider,
		tmNode.DefaultMetricsProvider(cfg.Instrumentation),
		log.With("module", "tendermint"),
	)

	if err != nil {
		log.Error(fmt.Sprintf("Failed to create a node: %v", err))
		os.Exit(1)
	}

	if err = node.Start(); err != nil {
		log.Error(fmt.Sprintf("Failed to start node: %v", err))
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
