package cmd

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/spf13/cobra"
	"github.com/tendermint/go-amino"
)

var VerifyGenesis = &cobra.Command{
	Use:   "verify_genesis",
	Short: "Verify genesis file",
	RunE:  verifyGenesis,
}

func verifyGenesis(cmd *cobra.Command, args []string) error {
	genesis, err := getGenesis()
	if err != nil {
		return err
	}

	var genesisState types.AppState
	if err := amino.UnmarshalJSON(genesis.AppState, &genesisState); err != nil {
		return err
	}

	if err := genesisState.Verify(); err != nil {
		return err
	}

	fmt.Printf("Genesis is ok\n")

	return nil
}
