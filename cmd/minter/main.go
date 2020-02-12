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
		cmd.Version)

	rootCmd.PersistentFlags().StringVar(&utils.MinterHome, "home-dir", "", "base dir (default is $HOME/.minter)")
	rootCmd.PersistentFlags().StringVar(&utils.MinterConfig, "config", "", "path to config (default is $(home-dir)/config/config.toml)")
	rootCmd.PersistentFlags().Bool("pprof", false, "enable pprof")
	rootCmd.PersistentFlags().String("pprof-addr", "0.0.0.0:6060", "pprof listen addr")

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
