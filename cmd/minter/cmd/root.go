package cmd

import (
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
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
		homeDir, err := cmd.Flags().GetString("home-dir")
		if err != nil {
			panic(err)
		}
		configDir, err := cmd.Flags().GetString("config")
		if err != nil {
			panic(err)
		}
		storage := utils.NewStorage(homeDir, configDir)
		v.SetConfigFile(storage.GetMinterConfigPath())
		cfg = config.GetConfig(storage.GetMinterHome())

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

		peers, err := cmd.Flags().GetString("persistent-peers")
		if err != nil {
			panic(err)
		}

		if peers != "" {
			cfg.P2P.PersistentPeers = peers
		}

		stateSync, err := cmd.Flags().GetBool("state-sync")
		if err != nil {
			panic(err)
		}

		if stateSync {
			cfg.StateSync.Enable = stateSync
		}
	},
}
