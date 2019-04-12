package cmd

import (
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/spf13/cobra"
)

var cfg = config.GetConfig()

var RootCmd = &cobra.Command{
	Use:   "minter",
	Short: "Minter Go Node",
}
