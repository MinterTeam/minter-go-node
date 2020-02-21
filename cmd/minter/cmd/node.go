package cmd

import (
	"context"
	api_v1 "github.com/MinterTeam/minter-go-node/api"
	api_v2 "github.com/MinterTeam/minter-go-node/api/v2"
	service_api "github.com/MinterTeam/minter-go-node/api/v2/service"
	"github.com/MinterTeam/minter-go-node/cli/service"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/statistics"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/gobuffalo/packr"
	"github.com/spf13/cobra"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	rpc "github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	"net/http"
	_ "net/http/pprof"
	"os"
)

var RunNode = &cobra.Command{
	Use:   "node",
	Short: "Run the Minter node",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNode(cmd)
	},
}

func runNode(cmd *cobra.Command) error {
	logger := log.NewLogger(cfg)

	pprofOn, err := cmd.Flags().GetBool("pprof")
	if err != nil {
		return err
	}

	if pprofOn {
		pprofAddr, err := cmd.Flags().GetString("pprof-addr")
		if err != nil {
			return err
		}

		pprofMux := http.DefaultServeMux
		http.DefaultServeMux = http.NewServeMux()
		go func() {
			logger.Error((&http.Server{
				Addr:    pprofAddr,
				Handler: pprofMux,
			}).ListenAndServe().Error())
		}()
	}

	tmConfig := config.GetTmConfig(cfg)

	if err := tmos.EnsureDir(utils.GetMinterHome()+"/config", 0777); err != nil {
		return err
	}

	if err := tmos.EnsureDir(utils.GetMinterHome()+"/tmdata", 0777); err != nil {
		return err
	}

	if cfg.KeepLastStates < 1 {
		panic("keep_last_states field should be greater than 0")
	}

	app := minter.NewMinterBlockchain(cfg)

	// update BlocksTimeDelta in case it was corrupted
	updateBlocksTimeDelta(app, tmConfig)

	// start TM node
	node := startTendermintNode(app, tmConfig, logger)

	client := rpc.NewLocal(node)

	app.SetTmNode(node)

	if !cfg.ValidatorMode {
		go func(srv *service_api.Service) {
			logger.Error("Failed to start Api V2 in both gRPC and RESTful", api_v2.Run(srv, cfg.GRPCListenAddress, cfg.APIv2ListenAddress))
		}(service_api.NewService(amino.NewCodec(), app, client, node, cfg, version.Version))

		go api_v1.RunAPI(app, client, cfg, logger)
	}

	ctx, stop := context.WithCancel(context.Background())
	ctxCli, _ := context.WithCancel(ctx)
	go func() {
		err := service.StartCLIServer(utils.GetMinterHome()+"/manager.sock", service.NewManager(app, client, cfg), ctxCli)
		if err != nil {
			panic(err)
		}
	}()
	//todo check ON/OFF
	app.InitStatistic()
	ctxStat, _ := context.WithCancel(ctx)
	go statistics.Statistic(ctxStat, app)
	tmos.TrapSignal(logger.With("module", "trap"), func() {
		// Cleanup
		stop()
		err := node.Stop()
		app.Stop()
		if err != nil {
			panic(err)
		}
	})

	// Run forever
	select {}
}

func updateBlocksTimeDelta(app *minter.Blockchain, config *tmCfg.Config) {
	blockStoreDB, err := tmNode.DefaultDBProvider(&tmNode.DBContext{ID: "blockstore", Config: config})
	if err != nil {
		panic(err)
	}

	blockStore := store.NewBlockStore(blockStoreDB)
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

func startTendermintNode(app types.Application, cfg *tmCfg.Config, logger tmlog.Logger) *tmNode.Node {
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
		logger.With("module", "tendermint"),
	)

	if err != nil {
		logger.Error("failed to create a node", "err", err)
		os.Exit(1)
	}

	if err = node.Start(); err != nil {
		logger.Error("failed to start node", "err", err)
		os.Exit(1)
	}

	logger.Info("Started node", "nodeInfo", node.Switch().NodeInfo())

	return node
}

func getGenesis() (doc *tmTypes.GenesisDoc, e error) {
	genesisFile := utils.GetMinterHome() + "/config/genesis.json"

	if !tmos.FileExists(genesisFile) {
		box := packr.NewBox("../../../genesis/current/")
		bytes, err := box.MustBytes("genesis.json")
		if err != nil {
			panic(err)
		}

		if err := tmos.WriteFile(genesisFile, bytes, 0644); err != nil {
			return nil, err
		}
	}

	return tmTypes.GenesisDocFromFile(genesisFile)
}
