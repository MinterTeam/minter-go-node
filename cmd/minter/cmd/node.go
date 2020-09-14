package cmd

import (
	"fmt"
	apiV1 "github.com/MinterTeam/minter-go-node/api"
	apiV2 "github.com/MinterTeam/minter-go-node/api/v2"
	serviceApi "github.com/MinterTeam/minter-go-node/api/v2/service"
	"github.com/MinterTeam/minter-go-node/cli/service"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/statistics"
	minterTypes "github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/spf13/cobra"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	tmLog "github.com/tendermint/tendermint/libs/log"
	tmOS "github.com/tendermint/tendermint/libs/os"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	"github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	"io"
	"net/http"
	_ "net/http/pprof" // nolint: gosec // securely exposed on separate, optional port
	"net/url"
	"os"
	"syscall"
)

// RunNode is the command that allows the CLI to start a node.
var RunNode = &cobra.Command{
	Use:   "node",
	Short: "Run the Minter node",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runNode(cmd)
	},
}

func runNode(cmd *cobra.Command) error {
	logger := log.NewLogger(cfg)

	// check open files limits
	if err := checkRlimits(); err != nil {
		panic(err)
	}

	// ensure /config and /tmdata dirs
	if err := ensureDirs(); err != nil {
		return err
	}

	isTestnet, _ := cmd.Flags().GetBool("testnet")
	if isTestnet {
		minterTypes.CurrentChainID = minterTypes.ChainTestnet
		version.Version += "-testnet"
	}

	pprofOn, err := cmd.Flags().GetBool("pprof")
	if err != nil {
		return err
	}

	if pprofOn {
		if err := enablePprof(cmd, logger); err != nil {
			return err
		}
	}

	tmConfig := config.GetTmConfig(cfg)

	app := minter.NewMinterBlockchain(cfg)

	// update BlocksTimeDelta in case it was corrupted
	updateBlocksTimeDelta(app, tmConfig)

	// start TM node
	node := startTendermintNode(app, tmConfig, logger)
	client := rpc.New(node)
	app.SetTmNode(node)

	if !cfg.ValidatorMode {
		runAPI(logger, app, client, node)
	}

	runCLI(cmd, app, client, node)

	if cfg.Instrumentation.Prometheus {
		go app.SetStatisticData(statistics.New()).Statistic(cmd.Context())
	}

	<-cmd.Context().Done()

	defer app.Stop()
	if err := node.Stop(); err != nil {
		return err
	}

	return nil
}

func runCLI(cmd *cobra.Command, app *minter.Blockchain, client *rpc.Local, tmNode *tmNode.Node) {
	go func() {
		err := service.StartCLIServer(utils.GetMinterHome()+"/manager.sock", service.NewManager(app, client, tmNode, cfg), cmd.Context())
		if err != nil {
			panic(err)
		}
	}()
}

func runAPI(logger tmLog.Logger, app *minter.Blockchain, client *rpc.Local, node *tmNode.Node) {
	go func(srv *serviceApi.Service) {
		grpcURL, err := url.Parse(cfg.GRPCListenAddress)
		if err != nil {
			logger.Error("Failed to parse gRPC address", err)
		}
		apiV2url, err := url.Parse(cfg.APIv2ListenAddress)
		if err != nil {
			logger.Error("Failed to parse API v2 address", err)
		}
		traceLog := os.Getenv("API_V2_LOG_LEVEL") == "trace"
		logger.Error("Failed to start Api V2 in both gRPC and RESTful",
			apiV2.Run(srv, grpcURL.Host, apiV2url.Host, traceLog))
	}(serviceApi.NewService(amino.NewCodec(), app, client, node, cfg, version.Version))

	go apiV1.RunAPI(app, client, cfg, logger)
}

func enablePprof(cmd *cobra.Command, logger tmLog.Logger) error {
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
	return nil
}

func ensureDirs() error {
	if err := tmOS.EnsureDir(utils.GetMinterHome()+"/config", 0777); err != nil {
		return err
	}

	if err := tmOS.EnsureDir(utils.GetMinterHome()+"/tmdata", 0777); err != nil {
		return err
	}

	return nil
}

func checkRlimits() error {
	const RequiredOpenFilesLimit = 10000

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}

	required := RequiredOpenFilesLimit + uint64(cfg.StateMemAvailable)
	if rLimit.Cur < required {
		rLimit.Cur = required
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			return fmt.Errorf("cannot set RLIMIT_NOFILE to %d", rLimit.Cur)
		}
	}

	return nil
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

func startTendermintNode(app types.Application, cfg *tmCfg.Config, logger tmLog.Logger) *tmNode.Node {
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
	genDocFile := utils.GetMinterHome() + "/config/genesis.json"
	_, err := os.Stat(genDocFile)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		if err := downloadFile(genDocFile, "https://raw.githubusercontent.com/MinterTeam/minter-network-migrate/master/minter-mainnet-2/genesis.json"); err != nil {
			panic(err)
		}
	}
	return tmTypes.GenesisDocFromFile(genDocFile)
}

func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
