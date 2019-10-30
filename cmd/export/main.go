package main

import (
	"encoding/json"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/appdb"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tm-db"
	"time"
)

func main() {
	err := common.EnsureDir(utils.GetMinterHome()+"/config", 0777)
	if err != nil {
		panic(err)
	}

	ldb, err := db.NewGoLevelDB("state", utils.GetMinterHome()+"/data")
	if err != nil {
		panic(err)
	}

	applicationDB := appdb.NewAppDB(config.GetConfig())
	height := applicationDB.GetLastHeight()
	currentState, err := state.NewState(height, ldb, nil, nil, 0, 100000)
	if err != nil {
		panic(err)
	}

	cdc := amino.NewCodec()

	jsonBytes, err := cdc.MarshalJSONIndent(currentState.Export(height), "", "	")
	if err != nil {
		panic(err)
	}

	appHash := [32]byte{}

	// Compose Genesis
	genesis := types.GenesisDoc{
		GenesisTime: time.Date(2019, time.April, 2, 17, 0, 0, 0, time.UTC),
		ChainID:     "minter-test-network-35",
		ConsensusParams: &types.ConsensusParams{
			Block: types.BlockParams{
				MaxBytes:   10000000,
				MaxGas:     100000,
				TimeIotaMs: 1000,
			},
			Evidence: types.EvidenceParams{
				MaxAge: 1000,
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
}
