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
		cmd.Manager,
		cmd.Version)

	rootCmd.PersistentFlags().StringVar(&utils.MinterHome, "home-dir", "", "base dir (default is $HOME/.minter)")
	rootCmd.PersistentFlags().StringVar(&utils.MinterConfig, "config", "", "path to config (default is $(home-dir)/config/config.toml)")

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
