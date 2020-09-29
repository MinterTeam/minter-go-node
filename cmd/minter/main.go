package main

import (
	"context"
	"github.com/MinterTeam/minter-go-node/cmd/minter/cmd"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	"os"
)

func main() {
	rootCmd := cmd.RootCmd

	ctx, cancel := context.WithCancel(context.Background())

	tmos.TrapSignal(log.NewTMLogger(os.Stdout).With("module", "consensus"), func() {
		cancel()
	})

	rootCmd.AddCommand(
		cmd.RunNode,
		cmd.ShowNodeId,
		cmd.ShowValidator,
		cmd.ManagerCommand,
		cmd.ManagerConsole,
		cmd.VerifyGenesis,
		cmd.Version,
		cmd.ExportCommand,
	)

	rootCmd.PersistentFlags().StringVar(&utils.MinterHome, "home-dir", "", "base dir (default is $HOME/.minter)")
	rootCmd.PersistentFlags().StringVar(&utils.MinterConfig, "config", "", "path to config (default is $(home-dir)/config/config.toml)")
	rootCmd.PersistentFlags().Bool("testnet", false, "use \"true\" for testnet, mainnet is default")
	rootCmd.PersistentFlags().Bool("pprof", false, "enable pprof")
	rootCmd.PersistentFlags().String("pprof-addr", "0.0.0.0:6060", "pprof listen addr")

	cmd.ExportCommand.Flags().Uint64("height", 0, "export height")
	cmd.ExportCommand.Flags().Uint64("start-height", 0, "height for starting a new chain")
	cmd.ExportCommand.Flags().Bool("indent", false, "using indent")
	cmd.ExportCommand.Flags().String("chain-id", "", "export chain id")
	cmd.ExportCommand.Flags().Duration("genesis-time", 0, "export height")

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		panic(err)
	}
}
