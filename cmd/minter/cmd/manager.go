package cmd

import (
	"github.com/MinterTeam/minter-go-node/cli/service"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/spf13/cobra"
)

var ManagerCommand = &cobra.Command{
	Use:                "manager",
	Short:              "Minter manager execute command",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		console, err := service.ConfigureManagerConsole(utils.GetMinterHome() + "/manager.sock")
		if err != nil {
			return nil
		}
		return console.Execute(args)
	},
}

var ManagerConsole = &cobra.Command{
	Use:                "console",
	Short:              "Minter CLI manager",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		console, err := service.ConfigureManagerConsole(utils.GetMinterHome() + "/manager.sock")
		if err != nil {
			return nil
		}
		console.Cli()
		return nil
	},
}
