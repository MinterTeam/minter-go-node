package cmd

import (
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfg *config.Config

var RootCmd = &cobra.Command{
	Use:   "minter",
	Short: "Minter Go Node",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.SetConfigName("config")
		viper.SetConfigFile(utils.GetMinterConfigPath())

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil {
			// stderr, so if we redirect output to json file, this doesn't appear
			// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// ignore not found error, return other errors
			panic(err)
		}

		cfg = config.GetConfig()
	},
}
