package cmd

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/spf13/cobra"
	"github.com/tendermint/go-amino"
	cryptoAmino "github.com/tendermint/tendermint/crypto/encoding/amino"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/privval"
	"os"
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
	logger := log.NewLogger(cfg)
	if !tmos.FileExists(keyFilePath) {
		logger.Error("private validator file does not exist", "file", keyFilePath)
		os.Exit(1)
	}

	pv := privval.LoadFilePV(keyFilePath, cfg.PrivValidatorStateFile())
	fmt.Printf("Mp%x\n", pv.GetPubKey().Bytes()[5:])
	return nil
}
