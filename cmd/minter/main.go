package main

import (
	"github.com/MinterTeam/minter-go-node/cmd/minter/cmd"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
)

func main() {
	rootCmd := cmd.RootCmd

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
	rootCmd.PersistentFlags().Bool("pprof", false, "enable pprof")
	rootCmd.PersistentFlags().String("pprof-addr", "0.0.0.0:6060", "pprof listen addr")

	cmd.ExportCommand.Flags().Uint64("height", 0, "export height")
	cmd.ExportCommand.Flags().String("chain_id", "", "export chain id")
	cmd.ExportCommand.Flags().Duration("genesis_time", 0, "export height")

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
