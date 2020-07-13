package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/spf13/cobra"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/types"
	db "github.com/tendermint/tm-db"
	"io"
	"log"
	"os"
	"time"
)

var (
	ExportCommand = &cobra.Command{
		Use:   "export",
		Short: "Minter export command",
		RunE:  export,
	}
)

const (
	genesisPath = "genesis.json"

	blockMaxBytes   int64 = 10000000
	blockMaxGas     int64 = 100000
	blockTimeIotaMs int64 = 1000

	evidenceMaxAgeNumBlocks = 1000
	evidenceMaxAgeDuration  = 24 * time.Hour
)

func export(cmd *cobra.Command, args []string) error {
	height, err := cmd.Flags().GetUint64("height")
	if err != nil {
		log.Panicf("Cannot parse height: %s", err)
	}

	chainID, err := cmd.Flags().GetString("chain_id")
	if err != nil {
		log.Panicf("Cannot parse chain id: %s", err)
	}

	genesisTime, err := cmd.Flags().GetDuration("genesis_time")
	if err != nil {
		log.Panicf("Cannot parse genesis time: %s", err)
	}

	fmt.Println("Start exporting...")

	ldb, err := db.NewGoLevelDB("state", utils.GetMinterHome()+"/data")
	if err != nil {
		log.Panicf("Cannot load db: %s", err)
	}

	currentState, err := state.NewState(height, ldb, nil, 1, 1)
	if err != nil {
		log.Panicf("Cannot new state at given height: %s", err)
	}

	exportTimeStart, newState := time.Now(), currentState.Export11To12(height)
	fmt.Printf("State has been exported. Took %s", time.Since(exportTimeStart))

	jsonBytes, err := amino.NewCodec().MarshalJSONIndent(newState, "", "	")
	if err != nil {
		log.Panicf("Cannot marshal state to json: %s", err)
	}

	appHash := [32]byte{}

	// compose genesis
	genesis := types.GenesisDoc{
		GenesisTime: time.Unix(0, 0).Add(genesisTime),
		ChainID:     chainID,
		ConsensusParams: &types.ConsensusParams{
			Block: types.BlockParams{
				MaxBytes:   blockMaxBytes,
				MaxGas:     blockMaxGas,
				TimeIotaMs: blockTimeIotaMs,
			},
			Evidence: types.EvidenceParams{
				MaxAgeNumBlocks: evidenceMaxAgeNumBlocks,
				MaxAgeDuration:  evidenceMaxAgeDuration,
			},
			Validator: types.ValidatorParams{
				PubKeyTypes: []string{
					types.ABCIPubKeyTypeEd25519,
				},
			},
		},
		AppHash:  appHash[:],
		AppState: json.RawMessage(jsonBytes),
	}

	err = genesis.ValidateAndComplete()
	if err != nil {
		log.Panicf("Failed to validate: %s", err)
	}

	if err := genesis.SaveAs(genesisPath); err != nil {
		log.Panicf("Failed to save genesis file: %s", err)
	}

	hash := getFileSha256Hash(genesisPath)
	fmt.Printf("\nOK\n%x\n", hash)

	return nil
}

func getFileSha256Hash(file string) []byte {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return h.Sum(nil)
}
