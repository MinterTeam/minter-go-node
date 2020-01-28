package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tm-db"
	"golang.org/x/crypto/sha3"
	"log"
	"time"
)

var (
	height      = flag.Uint64("height", 0, "height")
	chainID     = flag.String("chain_id", "", "chain_id")
	genesisTime = flag.Duration("genesis_time", 0, "genesis_time")
)

func main() {
	required := []string{"height", "chain_id", "genesis_time"}
	flag.Parse()
	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			log.Fatalf("missing required --%s argument/flag\n", req)
		}
	}

	err := os.EnsureDir(utils.GetMinterHome()+"/config", 0777)
	if err != nil {
		panic(err)
	}

	ldb, err := db.NewGoLevelDB("state", utils.GetMinterHome()+"/data")
	if err != nil {
		panic(err)
	}

	currentState, err := state.NewState(*height, ldb, nil, 0, 100000)
	if err != nil {
		panic(err)
	}

	cdc := amino.NewCodec()

	jsonBytes, err := cdc.MarshalJSONIndent(currentState.Export(*height), "", "	")
	if err != nil {
		panic(err)
	}

	appHash := [32]byte{}

	// Compose Genesis
	genesis := types.GenesisDoc{
		GenesisTime: time.Unix(0, 0).Add(*genesisTime),
		ChainID:     *chainID,
		ConsensusParams: &types.ConsensusParams{
			Block: types.BlockParams{
				MaxBytes:   10000000,
				MaxGas:     100000,
				TimeIotaMs: 1000,
			},
			Evidence: types.EvidenceParams{
				MaxAgeNumBlocks: 1000,
			},
			Validator: types.ValidatorParams{
				PubKeyTypes: []string{types.ABCIPubKeyTypeEd25519},
			},
		},
		AppHash:  appHash[:],
		AppState: json.RawMessage(jsonBytes),
	}

	err = genesis.ValidateAndComplete()
	if err != nil {
		panic(err)
	}

	if err := genesis.SaveAs("genesis.json"); err != nil {
		panic(err)
	}

	fmt.Println("OK")
	fmt.Println(sha3.Sum512([]byte(fmt.Sprintf("%v", genesis))))
}
