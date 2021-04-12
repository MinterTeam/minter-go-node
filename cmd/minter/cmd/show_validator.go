package cmd

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/spf13/cobra"
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
	keyFilePath := cfg.PrivValidatorKeyFile()
	logger := log.NewLogger(cfg)
	if !tmos.FileExists(keyFilePath) {
		logger.Error("private validator file does not exist", "file", keyFilePath)
		os.Exit(1)
	}

	pv := privval.LoadFilePV(keyFilePath, cfg.PrivValidatorStateFile())
	key, err := pv.GetPubKey()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Mp%x\n", key.Bytes()[:])
	return nil
}
