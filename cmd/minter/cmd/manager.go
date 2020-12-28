package cmd

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/cli/service"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var ManagerCommand = &cobra.Command{
	Use:                "manager",
	Short:              "Minter manager execute command",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		newArgs := setParentFlags(cmd, args)
		homeDir, err := cmd.Flags().GetString("home-dir")
		if err != nil {
			return err
		}
		configDir, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}
		console, err := service.NewCLI(utils.NewStorage(homeDir, configDir).GetMinterHome() + "/manager.sock")
		if err != nil {
			return nil
		}

		err = console.Execute(newArgs)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		return nil
	},
}

var ManagerConsole = &cobra.Command{
	Use:                "console",
	Short:              "Minter CLI manager",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = setParentFlags(cmd, args)
		homeDir, err := cmd.Flags().GetString("home-dir")
		if err != nil {
			return err
		}
		configDir, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}
		console, err := service.NewCLI(utils.NewStorage(homeDir, configDir).GetMinterHome() + "/manager.sock")
		if err != nil {
			return nil
		}
		err = console.Cli(cmd.Context())
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}

func setParentFlags(cmd *cobra.Command, args []string) (newArgs []string) {
	for _, arg := range args {
		split := strings.Split(arg, "=")
		if len(split) == 2 {
			err := cmd.Parent().PersistentFlags().Set(strings.TrimLeft(split[0], "-"), split[1])
			if err == nil {
				continue
			}
		}
		newArgs = append(newArgs, arg)
	}
	return newArgs
}
