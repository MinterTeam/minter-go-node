package api

import (
	"encoding/json"
	"github.com/tendermint/tendermint/crypto"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/types"
	"time"
)

// Genesis file
type ResultGenesis struct {
	Genesis *GenesisDoc `json:"genesis"`
}

// GenesisValidator is an initial validator.
type GenesisValidator struct {
	Address types.Address `json:"address"`
	PubKey  crypto.PubKey `json:"public_key"`
	Power   int64         `json:"power"`
	Name    string        `json:"name"`
}

// GenesisDoc defines the initial conditions for a tendermint blockchain, in particular its validator set.
type GenesisDoc struct {
	GenesisTime     time.Time              `json:"genesis_time"`
	ChainID         string                 `json:"chain_id"`
	ConsensusParams *types.ConsensusParams `json:"consensus_params,omitempty"`
	Validators      []GenesisValidator     `json:"validators,omitempty"`
	AppHash         tmbytes.HexBytes       `json:"app_hash"`
	AppState        json.RawMessage        `json:"app_state,omitempty"`
}

func Genesis() (*ResultGenesis, error) {
	result, err := client.Genesis()
	if err != nil {
		return nil, err
	}

	return &ResultGenesis{
		Genesis: &GenesisDoc{
			GenesisTime: result.Genesis.GenesisTime,
			ChainID:     result.Genesis.ChainID,
			ConsensusParams: &types.ConsensusParams{
				Block: types.BlockParams{
					MaxBytes:   result.Genesis.ConsensusParams.Block.MaxBytes,
					MaxGas:     result.Genesis.ConsensusParams.Block.MaxGas,
					TimeIotaMs: result.Genesis.ConsensusParams.Block.TimeIotaMs,
				},
				Evidence: types.EvidenceParams{
					MaxAgeNumBlocks: result.Genesis.ConsensusParams.Evidence.MaxAgeNumBlocks,
					MaxAgeDuration:  result.Genesis.ConsensusParams.Evidence.MaxAgeDuration,
				},
				Validator: types.ValidatorParams{
					PubKeyTypes: result.Genesis.ConsensusParams.Validator.PubKeyTypes,
				},
			},
			AppHash:  result.Genesis.AppHash,
			AppState: result.Genesis.AppState,
		},
	}, nil
}
