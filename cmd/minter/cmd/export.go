package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/coreV2/minter"
	"github.com/MinterTeam/minter-go-node/coreV2/rewards"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/tendermint/go-amino"
	"io"
	"log"
	"os"
	"time"

	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/coreV2/appdb"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	mtypes "github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/spf13/cobra"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/types"
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

	chainID, err := cmd.Flags().GetString("chain-id")
	if err != nil {
		log.Panicf("Cannot parse chain id: %s", err)
	}

	genesisTime, err := cmd.Flags().GetDuration("genesis-time")
	if err != nil {
		log.Panicf("Cannot parse genesis time: %s", err)
	}

	indent, err := cmd.Flags().GetBool("indent")
	if err != nil {
		log.Panicf("Cannot parse indent: %s", err)
	}

	log.Println("Start exporting...")

	homeDir, err := cmd.Flags().GetString("home-dir")
	if err != nil {
		return err
	}
	storages := utils.NewStorage(homeDir, "")

	ldb, err := storages.InitStateLevelDB("data/state", nil)
	if err != nil {
		log.Panicf("Cannot load db: %s", err)
	}

	db := appdb.NewAppDB(storages.GetMinterHome(), cfg)

	currentState, err := state.NewCheckStateAtHeightV3(height, ldb)
	if err != nil {
		log.Panicf("Cannot new state at given height: %s, last available height %d", err, db.GetLastHeight())
	}

	exportTimeStart := time.Now()
	appState := currentState.Export()
	log.Printf("State has been exported. Took %s\n", time.Since(exportTimeStart))

	if err := appState.Verify(); err != nil {
		log.Fatalf("Failed to validate: %s\n", err)
	}
	log.Printf("Verify state OK\n")

	appState.Version = minter.V3
	//versions := db.GetVersions()
	//for _, v := range versions {
	//	appState.Versions = append(appState.Versions, mtypes.Version{
	//		Height: v.Height,
	//		Name:   v.Name,
	//	})
	//}

	//appState.Emission = db.Emission().String()
	appState.Emission = rewards.NewReward().GetBeforeBlock(height).String()
	reserve0, reserve1 := currentState.Swap().GetSwapper(0, 1993).Reserves()
	db.UpdatePriceBug(time.Unix(0, int64(genesisTime)).UTC(), reserve0, reserve1)
	t, r0, r1, reward, off := db.GetPrice()
	appState.PrevReward = mtypes.RewardPrice{
		Time:       uint64(t.UTC().UnixNano()),
		AmountBIP:  r0.String(),
		AmountUSDT: r1.String(),
		Off:        off,
		Reward:     reward.String(),
	}
	var jsonBytes []byte
	if indent {
		jsonBytes, err = amino.NewCodec().MarshalJSONIndent(appState, "", "	")
	} else {
		jsonBytes, err = amino.NewCodec().MarshalJSON(appState)
	}
	if err != nil {
		log.Panicf("Cannot marshal state to json: %s", err)
	}
	log.Printf("Marshal OK\n")

	// compose genesis
	genesis := types.GenesisDoc{
		GenesisTime:   time.Unix(0, 0).Add(genesisTime),
		InitialHeight: int64(height),
		ChainID:       chainID,
		ConsensusParams: &tmproto.ConsensusParams{
			Block: tmproto.BlockParams{
				MaxBytes:   blockMaxBytes,
				MaxGas:     blockMaxGas,
				TimeIotaMs: blockTimeIotaMs,
			},
			Evidence: tmproto.EvidenceParams{
				MaxAgeNumBlocks: evidenceMaxAgeNumBlocks,
				MaxAgeDuration:  evidenceMaxAgeDuration,
			},
			Validator: tmproto.ValidatorParams{
				PubKeyTypes: []string{
					types.ABCIPubKeyTypeEd25519,
				},
			},
			Version: tmproto.VersionParams{
				AppVersion: version.AppVer,
			},
		},
		AppHash: nil,
		//AppHash:  db.GetLastBlockHash(),
		AppState: json.RawMessage(jsonBytes),
	}

	err = genesis.ValidateAndComplete()
	if err != nil {
		log.Panicf("Failed to validate: %s", err)
	}
	log.Printf("Validate genesis OK\n")

	if err := genesis.SaveAs(genesisPath); err != nil {
		log.Panicf("Failed to save genesis file: %s", err)
	}

	hash := getFileSha256Hash(genesisPath)
	log.Printf("Finish with sha256 hash: \n%x\n", hash)

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
