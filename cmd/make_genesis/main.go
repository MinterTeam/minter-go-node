package main

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/core/developers"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	tmTypes "github.com/tendermint/tendermint/types"
	"math/big"
	"os"
	"sort"
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

	file, err := os.Open("cmd/make_genesis/data.csv")
	if err != nil {
		panic(err)
	}

	firstBalances := map[string]*big.Int{}
	secondBalances := map[string]*big.Int{}
	bonusBalances := map[string]*big.Int{}
	airdropBalances := map[string]*big.Int{}

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 5
	rawCSVdata, err := reader.ReadAll()

	p := big.NewFloat(0).SetInt(big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))

	for _, line := range rawCSVdata {
		role, address, balMain, balBonus, balAirdrop := line[0], line[1], line[2], line[3], line[4]
		if _, has := firstBalances[address]; !has {
			firstBalances[address] = big.NewInt(0)
			secondBalances[address] = big.NewInt(0)
		}

		if _, has := bonusBalances[address]; !has {
			bonusBalances[address] = big.NewInt(0)
		}

		if _, has := airdropBalances[address]; !has {
			airdropBalances[address] = big.NewInt(0)
		}

		balMainFloat, _ := big.NewFloat(0).SetString(balMain)
		balMainInt, _ := big.NewFloat(0).Mul(balMainFloat, p).Int(nil)
		if balMainInt.Cmp(helpers.BipToPip(big.NewInt(1000000))) != -1 || role == "pool_admin" {
			firstBalances[address].Add(firstBalances[address], balMainInt)
		} else {
			secondBalances[address].Add(secondBalances[address], balMainInt)
		}

		balBonusFloat, _ := big.NewFloat(0).SetString(balBonus)
		balBonusInt, _ := big.NewFloat(0).Mul(balBonusFloat, p).Int(nil)
		bonusBalances[address].Add(bonusBalances[address], balBonusInt)

		balAirdropFloat, _ := big.NewFloat(0).SetString(balAirdrop)
		balAirdropInt, _ := big.NewFloat(0).Mul(balAirdropFloat, p).Int(nil)
		airdropBalances[address].Add(airdropBalances[address], balAirdropInt)
	}

	var frozenFunds []types.FrozenFund

	for address, balance := range secondBalances {
		if balance.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		frozenFunds = append(frozenFunds, types.FrozenFund{
			Height:       17280 * 8,
			Address:      types.HexToAddress(address),
			CandidateKey: []byte{0},
			Coin:         types.GetBaseCoin(),
			Value:        balance,
		})
	}

	for address, balance := range bonusBalances {
		if balance.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		frozenFunds = append(frozenFunds, types.FrozenFund{
			Height:       17280 * 15,
			Address:      types.HexToAddress(address),
			CandidateKey: []byte{0},
			Coin:         types.GetBaseCoin(),
			Value:        balance,
		})
	}

	for address, balance := range airdropBalances {
		if balance.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		frozenFunds = append(frozenFunds, types.FrozenFund{
			Height:       17280 * 29,
			Address:      types.HexToAddress(address),
			CandidateKey: []byte{0},
			Coin:         types.GetBaseCoin(),
			Value:        balance,
		})
	}

	sort.SliceStable(frozenFunds, func(i, j int) bool {
		if frozenFunds[i].Height != frozenFunds[j].Height {
			return frozenFunds[i].Height < frozenFunds[j].Height
		}

		return frozenFunds[i].Address.Compare(frozenFunds[j].Address) == -1
	})

	sort.SliceStable(frozenFunds, func(i, j int) bool {
		if frozenFunds[i].Height != frozenFunds[j].Height {
			return frozenFunds[i].Height < frozenFunds[j].Height
		}

		return frozenFunds[i].Address.Compare(frozenFunds[j].Address) == -1
	})

	bals := makeBalances(firstBalances)

	sort.SliceStable(bals, func(i, j int) bool {
		return bals[i].Address.Compare(bals[j].Address) == -1
	})

	validators, candidates := makeValidatorsAndCandidates(validatorsPubKeys, big.NewInt(1))

	jsonBytes, err := cdc.MarshalJSONIndent(types.AppState{
		Validators:   validators,
		Candidates:   candidates,
		Accounts:     bals,
		MaxGas:       minter.DefaultMaxGas,
		TotalSlashed: big.NewInt(0),
		FrozenFunds:  frozenFunds,
	}, "", "	")
	if err != nil {
		panic(err)
	}

	appHash := [32]byte{}
	networkId := "minter-test-network-38"

	// Compose Genesis
	genesis := tmTypes.GenesisDoc{
		GenesisTime: time.Date(2019, time.April, 15, 17, 0, 0, 0, time.UTC),
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

func makeBalances(balances map[string]*big.Int) []types.Account {
	totalBalances := big.NewInt(0)
	for _, val := range balances {
		totalBalances.Add(totalBalances, val)
	}

	balances[developers.Address.String()] = big.NewInt(0).Sub(helpers.BipToPip(big.NewInt(200000000)), totalBalances) // Developers account

	var result []types.Account
	for address, balance := range balances {
		if balance.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		result = append(result, types.Account{
			Address: types.HexToAddress(address),
			Balance: []types.Balance{
				{
					Coin:  types.GetBaseCoin(),
					Value: balance,
				},
			},
		})
	}

	return result
}
