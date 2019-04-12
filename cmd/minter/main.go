package main

import (
	"github.com/MinterTeam/minter-go-node/cmd/minter/cmd"
)

func main() {
	rootCmd := cmd.RootCmd

	rootCmd.AddCommand(
		cmd.RunNode,
		cmd.ShowNodeId,
		cmd.ShowValidator)

	if err := cmd.RootCmd.Execute(); err != nil {
		panic(err)
	}
}
