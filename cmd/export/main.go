package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/cmd/export/types11"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/state"
	mtypes "github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/types"
	"io"
	"log"
	"os"
	"time"
)

var (
	height      = flag.Uint64("height", 0, "height")
	chainID     = flag.String("chain_id", "", "chain_id")
	genesisTime = flag.Duration("genesis_time", 0, "genesis_time")
)

const (
	genesisPath = "genesis.json"

	maxSupply = "1000000000000000000000000000000000"
)

func main() {
	flag.Parse()

	required := []string{"height", "chain_id", "genesis_time"}
	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			log.Fatalf("missing required --%s argument/flag\n", req)
		}
	}

	err := common.EnsureDir(utils.GetMinterHome()+"/config", 0777)
	if err != nil {
		panic(err)
	}

	ldb, err := db.NewGoLevelDB("state", utils.GetMinterHome()+"/data")
	if err != nil {
		panic(err)
	}

	currentState, err := state.New(*height, ldb, false)
	if err != nil {
		panic(err)
	}

	oldState := currentState.Export(*height)
	newState := types11.AppState{
		Note:         oldState.Note,
		StartHeight:  oldState.StartHeight,
		MaxGas:       oldState.MaxGas,
		TotalSlashed: oldState.TotalSlashed.String(),
	}

	for _, account := range oldState.Accounts {
		newState.Accounts = append(newState.Accounts, mtypes.Account{
			Address:      account.Address,
			Balance:      account.Balance,
			Nonce:        account.Nonce,
			MultisigData: account.MultisigData,
		})
	}

	for _, coin := range oldState.Coins {
		newState.Coins = append(newState.Coins, types11.Coin{
			Name:      coin.Name,
			Symbol:    coin.Symbol,
			Volume:    coin.Volume.String(),
			Crr:       coin.Crr,
			Reserve:   coin.ReserveBalance.String(),
			MaxSupply: maxSupply,
		})
	}

	for _, check := range oldState.UsedChecks {
		newState.UsedChecks = append(newState.UsedChecks, check)
	}

	for _, ff := range oldState.FrozenFunds {
		newState.FrozenFunds = append(newState.FrozenFunds, mtypes.FrozenFund{
			Height:       ff.Height,
			Address:      ff.Address,
			CandidateKey: ff.CandidateKey,
			Coin:         ff.Coin,
			Value:        ff.Value,
		})
	}

	for _, candidate := range oldState.Candidates {
		newState.Candidates = append(newState.Candidates, mtypes.Candidate{
			RewardAddress: candidate.RewardAddress,
			OwnerAddress:  candidate.OwnerAddress,
			TotalBipStake: candidate.TotalBipStake,
			PubKey:        candidate.PubKey,
			Commission:    candidate.Commission,
			Stakes:        candidate.Stakes,
			Status:        candidate.Status,
		})
	}

	for _, validator := range oldState.Validators {
		newState.Validators = append(newState.Validators, types11.Validator{
			TotalBipStake: validator.TotalBipStake.String(),
			PubKey:        validator.PubKey,
			AccumReward:   validator.AccumReward.String(),
			AbsentTimes:   validator.AbsentTimes,
		})
	}

	cdc := amino.NewCodec()
	jsonBytes, err := cdc.MarshalJSONIndent(newState, "", "	")
	if err != nil {
		panic(err)
	}

	appHash := [32]byte{}

	// Compose Genesis
	genesis := types11.GenesisDoc{
		GenesisTime: time.Unix(0, 0).Add(*genesisTime),
		ChainID:     *chainID,
		ConsensusParams: &types11.ConsensusParams{
			Block: types11.BlockParams{
				MaxBytes:   10000000,
				MaxGas:     100000,
				TimeIotaMs: 1000,
			},
			Evidence: types11.EvidenceParams{
				MaxAgeNumBlocks: 1000,
				MaxAgeDuration:  24 * time.Hour,
			},
			Validator: types11.ValidatorParams{
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

	if err := genesis.SaveAs(genesisPath); err != nil {
		panic(err)
	}

	fmt.Printf("Ok\n%x\n", getSha256Hash(genesisPath))
}

func getSha256Hash(file string) []byte {
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
