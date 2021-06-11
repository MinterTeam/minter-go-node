package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof" // nolint: gosec // securely exposed on separate, optional port
	"net/url"
	"os"
	"syscall"

	apiV2 "github.com/MinterTeam/minter-go-node/api/v2"
	serviceApi "github.com/MinterTeam/minter-go-node/api/v2/service"
	"github.com/MinterTeam/minter-go-node/cli/service"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/coreV2/minter"
	"github.com/MinterTeam/minter-go-node/coreV2/rewards"
	"github.com/MinterTeam/minter-go-node/coreV2/statistics"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/spf13/cobra"
	tmCfg "github.com/tendermint/tendermint/config"
	tmLog "github.com/tendermint/tendermint/libs/log"
	tmOS "github.com/tendermint/tendermint/libs/os"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	tmTypes "github.com/tendermint/tendermint/types"
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

	homeDir, err := cmd.Flags().GetString("home-dir")
	if err != nil {
		return err
	}
	configDir, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	storages := utils.NewStorage(homeDir, configDir)

	// ensure /config and /tmdata dirs
	if err := ensureDirs(storages.GetMinterHome()); err != nil {
		return err
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

	if !cfg.ValidatorMode {
		_, err = storages.InitEventLevelDB("data/events", minter.GetDbOpts(1024))
		if err != nil {
			return err
		}
	}
	_, err = storages.InitStateLevelDB("data/state", minter.GetDbOpts(cfg.StateMemAvailable))
	if err != nil {
		return err
	}
	app := minter.NewMinterBlockchain(storages, cfg, cmd.Context(), 0)

	// start TM node
	node := startTendermintNode(app, tmConfig, logger, storages.GetMinterHome())
	client := app.RpcClient()

	if !cfg.ValidatorMode {
		runAPI(logger, app, client, node, app.RewardCounter())
	}

	runCLI(cmd.Context(), app, client, node, storages.GetMinterHome())

	if cfg.Instrumentation.Prometheus {
		go app.SetStatisticData(statistics.New()).Statistic(cmd.Context())
	}

	return app.WaitStop()
}

func runCLI(ctx context.Context, app *minter.Blockchain, client *rpc.Local, tmNode *tmNode.Node, home string) {
	go func() {
		err := service.StartCLIServer(home+"/manager.sock", service.NewManager(app, client, tmNode, cfg), ctx)
		if err != nil {
			panic(err)
		}
	}()
}

func runAPI(logger tmLog.Logger, app *minter.Blockchain, client *rpc.Local, node *tmNode.Node, reward *rewards.Reward) {
	go func(srv *serviceApi.Service) {
		grpcURL, err := url.Parse(cfg.GRPCListenAddress)
		if err != nil {
			logger.Error("Failed to parse gRPC address", err)
		}
		apiV2url, err := url.Parse(cfg.APIv2ListenAddress)
		if err != nil {
			logger.Error("Failed to parse API v2 address", err)
		}
		logger.Error("Failed to start Api V2 in both gRPC and RESTful",
			apiV2.Run(srv, grpcURL.Host, apiV2url.Host, logger.With("module", "rpc")))
	}(serviceApi.NewService(app, client, node, cfg, version.Version, reward))
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

func ensureDirs(homeDir string) error {
	if err := tmOS.EnsureDir(homeDir+"/config", 0777); err != nil {
		return err
	}

	if err := tmOS.EnsureDir(homeDir+"/tmdata", 0777); err != nil {
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

func startTendermintNode(app *minter.Blockchain, cfg *tmCfg.Config, logger tmLog.Logger, home string) *tmNode.Node {
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		panic(err)
	}

	node, err := tmNode.NewNode(
		cfg,
		privval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(app),
		getGenesis(home+"/config/genesis.json"),
		tmNode.DefaultDBProvider,
		tmNode.DefaultMetricsProvider(cfg.Instrumentation),
		logger.With("module", "tendermint"),
	)

	if err != nil {
		logger.Error("failed to create a node", "err", err)
		os.Exit(1)
	}

	app.SetTmNode(node)

	if err = node.Start(); err != nil {
		logger.Error("failed to start node", "err", err)
		os.Exit(1)
	}

	logger.Info("Started node", "nodeInfo", node.Switch().NodeInfo())

	return node
}

func getGenesis(genDocFile string) func() (doc *tmTypes.GenesisDoc, e error) {
	return func() (doc *tmTypes.GenesisDoc, e error) {
		_, err := os.Stat(genDocFile)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}

			genesis, err := RootCmd.Flags().GetString("genesis")
			if err != nil {
				return nil, err
			}

			if err := downloadFile(genDocFile, genesis); err != nil {
				return nil, err
			}
		}
		doc, err = tmTypes.GenesisDocFromFile(genDocFile)
		if err != nil {
			return nil, err
		}
		if len(doc.AppHash) == 0 {
			doc.AppHash = nil
		}
		return doc, err
	}
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
