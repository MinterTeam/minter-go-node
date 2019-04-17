package main

import (
	"github.com/MinterTeam/minter-go-node/cmd/minter/cmd"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
)

func main() {
	rootCmd := cmd.RootCmd
	rootCmd.PersistentFlags().StringVar(&utils.MinterHome, "home-dir", "", "base dir (default is $HOME/.minter)")

	rootCmd.AddCommand(
		cmd.RunNode,
		cmd.ShowNodeId,
		cmd.ShowValidator)

	if err := cmd.RootCmd.Execute(); err != nil {
		panic(err)
	}
}
