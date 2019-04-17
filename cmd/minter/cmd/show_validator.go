package cmd

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/spf13/cobra"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/privval"
)

var ShowValidator = &cobra.Command{
	Use:   "show_validator",
	Short: "Show this node's validator public key",
	RunE:  showValidator,
}

func showValidator(cmd *cobra.Command, args []string) error {
	cdc := amino.NewCodec()
	cryptoAmino.RegisterAmino(cdc)

	keyFilePath := cfg.PrivValidatorKeyFile()
	if !common.FileExists(keyFilePath) {
		log.Fatal("private validator file does not exist", "file", keyFilePath)
	}

	pv := privval.LoadFilePV(keyFilePath, cfg.PrivValidatorStateFile())
	fmt.Printf("Mp%x\n", pv.GetPubKey().Bytes()[5:])
	return nil
}
