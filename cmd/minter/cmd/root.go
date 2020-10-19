package cmd

import (
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfg *config.Config

var RootCmd = &cobra.Command{
	Use:   "minter",
	Short: "Minter Go Node",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		v := viper.New()
		v.SetConfigFile(utils.GetMinterConfigPath())
		cfg = config.GetConfig()

		if err := v.ReadInConfig(); err != nil {
			panic(err)
		}

		if err := v.Unmarshal(cfg); err != nil {
			panic(err)
		}

		if cfg.KeepLastStates < 1 {
			panic("keep_last_states field should be greater than 0")
		}

		isTestnet, _ := cmd.Flags().GetBool("testnet")
		if isTestnet {
			types.CurrentChainID = types.ChainTestnet
			version.Version += "-testnet"
		}
	},
}
