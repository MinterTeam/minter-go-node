package cmd

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/api"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/gui"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/gobuffalo/packr"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/abci/types"
	bc "github.com/tendermint/tendermint/blockchain"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/common"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	rpc "github.com/tendermint/tendermint/rpc/client"
	tmTypes "github.com/tendermint/tendermint/types"
	"time"
)

var RunNode = &cobra.Command{
	Use:   "node",
	Short: "Run the Minter node",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNode()
	},
}

func runNode() error {
	now := time.Now()
	startTime := time.Date(2019, time.June, 5, 17, 0, 0, 0, time.UTC)
	if startTime.After(now) {
		fmt.Printf("Start time is in the future, sleeping until %s", startTime)
		time.Sleep(startTime.Sub(now))
	}

	tmConfig := config.GetTmConfig(cfg)

	if err := common.EnsureDir(utils.GetMinterHome()+"/config", 0777); err != nil {
		return err
	}

	if err := common.EnsureDir(utils.GetMinterHome()+"/tmdata", 0777); err != nil {
		return err
	}

	// init events db
	eventsdb.InitDB(cfg)

	app := minter.NewMinterBlockchain(cfg)

	// update BlocksTimeDelta in case it was corrupted
	updateBlocksTimeDelta(app, tmConfig)

	// start TM node
	node := startTendermintNode(app, tmConfig)

	client := rpc.NewLocal(node)
	status, _ := client.Status()
	if status.NodeInfo.Network != config.NetworkId {
		log.Fatal("Different networks", "expected", config.NetworkId, "got", status.NodeInfo.Network)
	}

	app.SetTmNode(node)

	if !cfg.ValidatorMode {
		go api.RunAPI(app, client, cfg)
		go gui.Run(cfg.GUIListenAddress)
	}

	// Recheck mempool. Currently kind a hack.
	go recheckMempool(node, cfg)

	common.TrapSignal(log.With("module", "trap"), func() {
		// Cleanup
		err := node.Stop()
		app.Stop()
		if err != nil {
			panic(err)
		}
	})

	// Run forever
	select {}
}

func recheckMempool(node *tmNode.Node, config *config.Config) {
	ticker := time.NewTicker(time.Minute)
	mempool := node.Mempool()
	for {
		select {
		case <-ticker.C:
			txs := mempool.ReapMaxTxs(config.Mempool.Size)
			mempool.Flush()

			for _, tx := range txs {
				_ = mempool.CheckTx(tx, func(res *types.Response) {})
			}
		}
	}
}

func updateBlocksTimeDelta(app *minter.Blockchain, config *tmCfg.Config) {
	blockStoreDB, err := tmNode.DefaultDBProvider(&tmNode.DBContext{ID: "blockstore", Config: config})
	if err != nil {
		panic(err)
	}

	blockStore := bc.NewBlockStore(blockStoreDB)
	height := uint64(blockStore.Height())
	count := uint64(3)
	if _, err := app.GetBlocksTimeDelta(height, count); height >= 20 && err != nil {
		blockA := blockStore.LoadBlockMeta(int64(height - count - 1))
		blockB := blockStore.LoadBlockMeta(int64(height - 1))

		delta := int(blockB.Header.Time.Sub(blockA.Header.Time).Seconds())
		app.SetBlocksTimeDelta(height, delta)
	}
	blockStoreDB.Close()
}

func startTendermintNode(app types.Application, cfg *tmCfg.Config) *tmNode.Node {
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		panic(err)
	}

	node, err := tmNode.NewNode(
		cfg,
		privval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(app),
		getGenesis,
		tmNode.DefaultDBProvider,
		tmNode.DefaultMetricsProvider(cfg.Instrumentation),
		log.With("module", "tendermint"),
	)

	if err != nil {
		log.Fatal("failed to create a node", "err", err)
	}

	if err = node.Start(); err != nil {
		log.Fatal("failed to start node", "err", err)
	}

	log.Info("Started node", "nodeInfo", node.Switch().NodeInfo())

	return node
}

func getGenesis() (doc *tmTypes.GenesisDoc, e error) {
	genesisFile := utils.GetMinterHome() + "/config/genesis.json"

	if !common.FileExists(genesisFile) {
		box := packr.NewBox("../../../testnet/")
		bytes, err := box.MustBytes(config.NetworkId + "/genesis.json")
		if err != nil {
			panic(err)
		}

		if err := common.WriteFile(genesisFile, bytes, 0644); err != nil {
			return nil, err
		}
	}

	return tmTypes.GenesisDocFromFile(genesisFile)
}
