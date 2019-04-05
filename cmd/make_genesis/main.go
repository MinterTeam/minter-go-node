package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/core/developers"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	tmTypes "github.com/tendermint/tendermint/types"
	"math/big"
	"time"
)

func main() {
	cdc := amino.NewCodec()

	validatorsPubKeys := []string{
		"SuHuc+YTbIWwypM6mhNHdYozSIXxCzI4OYpnrC6xU7g=",
		"c42kG6ant9abcpSvoVi4nFobQQy/DCRDyFxf4krR3Rw=",
		"bxbB/yGm+5RqrtD0wfzKJyty/ZBJiPkdOIMoK4rjG6I=",
		"nhPy9UaN14KzFkRPvWZZXhPbp9e9Pvob7NULQgRfWMY=",
	}

	balances := map[string]int64{
		"Mxe6732a97c6445edb0becf685dd92655bb4a1b838": 17318590, // Minter One
		"Mx89f5395a03847826d6b48bb02dbde64376945a20": 9833248,  // MonsterNode
		"Mx8da4f97b635cf751f2c0f0020f2e78ceb691d7d5": 3798414,  // BTC.Secure
		"Mxf5d006607e9420978b8f33a940a6fcbc67797db2": 2187460,  // Minter Store
		"Mx50003880d87db2fa48f6b824cdcfeeab1ac77733": 2038305,  // DeCenter
		"Mx198208eb4d11d4b389ff262ac52494d920770879": 1063320,  // MINTER CENTER
		"Mx3bdee0d64fa9ac892720f48724ef6a4e2919a6ba": 803817,   // Minternator
		"Mx8bee92ba8999ab047cb5e2d98e190dada4d7a2b2": 803118,   // StakeHolder
		"Mxe1dbde5c02a730f747a47d24f0f993c27da9dff1": 636837,   // Rundax
		"Mx601609b85ee21b9493dffbca1079c74d47b75f2a": 158857,   // PRO-BLOCKCHAIN
		"Mxdce154b6e1d06b46e95881b900eeb164e247c180": 75414,    // Mother Minter
		"Mxdc7fcc63930bf81ebdce12b3bcef57b93e99a157": 68258,    // Validator.Center
		"Mx0acbd5df9bc4bdc9fcf2f87e8393907739401a27": 34512,    // bipMaker
		"Mx35c40563ee5181899d0d605839edb9e940b0d8e5": 33869,    // SolidMinter
	}

	validators, candidates := makeValidatorsAndCandidates(validatorsPubKeys, big.NewInt(1))

	jsonBytes, err := cdc.MarshalJSONIndent(types.AppState{
		Validators:   validators,
		Candidates:   candidates,
		Accounts:     makeBalances(balances),
		MaxGas:       minter.DefaultMaxGas,
		TotalSlashed: big.NewInt(0),
	}, "", "	")
	if err != nil {
		panic(err)
	}

	appHash := [32]byte{}
	networkId := "minter-test-network-36"

	// Compose Genesis
	genesis := tmTypes.GenesisDoc{
		GenesisTime: time.Date(2019, time.April, 5, 17, 0, 0, 0, time.UTC),
		ChainID:     networkId,
		ConsensusParams: &tmTypes.ConsensusParams{
			Block: tmTypes.BlockParams{
				MaxBytes:   minter.BlockMaxBytes,
				MaxGas:     minter.DefaultMaxGas,
				TimeIotaMs: 1000,
			},
			Evidence: tmTypes.EvidenceParams{
				MaxAge: 1000,
			},
			Validator: tmTypes.ValidatorParams{
				PubKeyTypes: []string{tmTypes.ABCIPubKeyTypeEd25519},
			},
		},
		AppHash:  appHash[:],
		AppState: json.RawMessage(jsonBytes),
	}

	err = genesis.ValidateAndComplete()
	if err != nil {
		panic(err)
	}

	if err := genesis.SaveAs("testnet/" + networkId + "/genesis.json"); err != nil {
		panic(err)
	}
}

func makeValidatorsAndCandidates(pubkeys []string, stake *big.Int) ([]types.Validator, []types.Candidate) {
	validators := make([]types.Validator, len(pubkeys))
	candidates := make([]types.Candidate, len(pubkeys))
	addr := developers.Address

	for i, val := range pubkeys {
		pkey, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			panic(err)
		}

		validators[i] = types.Validator{
			RewardAddress: addr,
			TotalBipStake: stake,
			PubKey:        pkey,
			Commission:    100,
			AccumReward:   big.NewInt(0),
			AbsentTimes:   types.NewBitArray(24),
		}

		candidates[i] = types.Candidate{
			RewardAddress: addr,
			OwnerAddress:  addr,
			TotalBipStake: big.NewInt(1),
			PubKey:        pkey,
			Commission:    100,
			Stakes: []types.Stake{
				{
					Owner:    addr,
					Coin:     types.GetBaseCoin(),
					Value:    stake,
					BipValue: stake,
				},
			},
			CreatedAtBlock: 1,
			Status:         state.CandidateStatusOnline,
		}
	}

	return validators, candidates
}

func makeBalances(balances map[string]int64) []types.Account {
	var totalBalances int64
	for _, val := range balances {
		totalBalances += val
	}

	balances[developers.Address.String()] = 200000000 - totalBalances // Developers account

	result := make([]types.Account, len(balances))
	i := 0
	for address, balance := range balances {
		result[i] = types.Account{
			Address: types.HexToAddress(address),
			Balance: []types.Balance{
				{
					Coin:  types.GetBaseCoin(),
					Value: helpers.BipToPip(big.NewInt(balance)),
				},
			},
		}
		i++
	}

	return result
}
