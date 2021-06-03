package main

import (
	"context"
	"github.com/MinterTeam/minter-go-node/cmd/minter/cmd"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	"os"
	"time"
)

func main() {
	rootCmd := cmd.RootCmd

	ctx, cancel := context.WithCancel(context.Background())

	tmos.TrapSignal(log.NewTMLogger(os.Stdout).With("module", "consensus"), func() {
		cancel()
		time.Sleep(time.Second * 10)
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

	rootCmd.PersistentFlags().String("home-dir", "", "base dir (default is $HOME/.minter)")
	rootCmd.PersistentFlags().String("config", "", "path to config (default is $(home-dir)/config/config.toml)")
	rootCmd.PersistentFlags().Bool("testnet", false, "use \"true\" for testnet, mainnet is default")
	rootCmd.PersistentFlags().Bool("pprof", false, "enable pprof")
	rootCmd.PersistentFlags().String("pprof-addr", "0.0.0.0:6060", "pprof listen addr")
	rootCmd.PersistentFlags().String("genesis", "https://github.com/MinterTeam/minter-go-node/releases/download/v2.0/genesis.json", "path with the genesis file to download")

	cmd.ExportCommand.Flags().Uint64("height", 0, "export height")
	cmd.ExportCommand.Flags().Bool("indent", false, "using indent")
	cmd.ExportCommand.Flags().String("chain-id", "", "export chain id")
	cmd.ExportCommand.Flags().Duration("genesis-time", 0, "export height")

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		panic(err)
	}
}
