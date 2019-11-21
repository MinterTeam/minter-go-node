package cmd

import (
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-node-cli/service"
	"github.com/spf13/cobra"
)

var Manager = &cobra.Command{
	Use:                "manager",
	Short:              "Minter CLI manager",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		service.RunCli(utils.GetMinterHome()+"/manager.sock", args)
		return nil
	},
}
