package main

import (
	"github.com/MinterTeam/minter-go-node/cmd/minter/cmd"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
)

func main() {
	rootCmd := cmd.RootCmd

	rootCmd.AddCommand(
		cmd.RunNode,
		cmd.ShowNodeId,
		cmd.ShowValidator,
		cmd.Version)

	rootCmd.PersistentFlags().StringVar(&utils.MinterHome, "home-dir", "", "base dir (default is $HOME/.minter)")
	rootCmd.PersistentFlags().StringVar(&utils.MinterConfig, "config", "", "path to config (default is $(home-dir)/config/config.toml)")
	rootCmd.PersistentFlags().StringVar(&config.NetworkId, "network-id", config.DefaultNetworkId, "network id")

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
