package genesis

import (
	"encoding/base64"
	"encoding/json"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/core/developers"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	tmtypes "github.com/tendermint/tendermint/types"
	"math/big"
	"time"
)

var (
	Network     = "minter-test-network-34"
	genesisTime = time.Date(2019, time.March, 27, 12, 0, 0, 0, time.UTC)

	BlockMaxBytes int64 = 10000000
	DefaultMaxGas int64 = 100000
)

func GetTestnetGenesis() (*tmtypes.GenesisDoc, error) {
	validatorsPubKeys := []string{
		"SuHuc+YTbIWwypM6mhNHdYozSIXxCzI4OYpnrC6xU7g=",
		"c42kG6ant9abcpSvoVi4nFobQQy/DCRDyFxf4krR3Rw=",
		"bxbB/yGm+5RqrtD0wfzKJyty/ZBJiPkdOIMoK4rjG6I=",
		"nhPy9UaN14KzFkRPvWZZXhPbp9e9Pvob7NULQgRfWMY=",
	}

	balances := map[string]int64{
		"Mx50003880d87db2fa48f6b824cdcfeeab1ac77733": 15021622, // DeCenter
		"Mxe6732a97c6445edb0becf685dd92655bb4a1b838": 12644321, // Minter One
		"Mx89f5395a03847826d6b48bb02dbde64376945a20": 5038353,  // MonsterNode
		"Mx8da4f97b635cf751f2c0f0020f2e78ceb691d7d5": 3434147,  // BTC.Secure
		"Mx198208eb4d11d4b389ff262ac52494d920770879": 1063320,  // MINTER CENTER
		"Mxf5d006607e9420978b8f33a940a6fcbc67797db2": 1003783,  // Minter Store
		"Mx8bee92ba8999ab047cb5e2d98e190dada4d7a2b2": 813816,   // StakeHolder
		"Mx3bdee0d64fa9ac892720f48724ef6a4e2919a6ba": 803817,   // Minternator
		"Mxe1dbde5c02a730f747a47d24f0f993c27da9dff1": 636837,   // Rundax
		"Mx601609b85ee21b9493dffbca1079c74d47b75f2a": 165133,   // PRO-BLOCKCHAIN
		"Mxdce154b6e1d06b46e95881b900eeb164e247c180": 75414,    // Mother Minter
		"Mxdc7fcc63930bf81ebdce12b3bcef57b93e99a157": 61982,    // Validator.Center
		"Mx0acbd5df9bc4bdc9fcf2f87e8393907739401a27": 34512,    // bipMaker
		"Mx35c40563ee5181899d0d605839edb9e940b0d8e5": 33869,    // SolidMinter
	}

	validators, candidates := MakeValidatorsAndCandidates(validatorsPubKeys, big.NewInt(1))

	cdc := amino.NewCodec()

	// Prepare initial AppState
	appStateJSON, err := cdc.MarshalJSONIndent(types.AppState{
		Validators: validators,
		Candidates: candidates,
		Accounts:   makeBalances(balances),
		MaxGas:     100000,
	}, "", "	")
	if err != nil {
		return nil, err
	}

	appHash := [32]byte{}

	// Compose Genesis
	genesis := tmtypes.GenesisDoc{
		GenesisTime: genesisTime,
		ChainID:     Network,
		ConsensusParams: &tmtypes.ConsensusParams{
			Block: tmtypes.BlockParams{
				MaxBytes:   BlockMaxBytes,
				MaxGas:     DefaultMaxGas,
				TimeIotaMs: 1000,
			},
			Evidence: tmtypes.EvidenceParams{
				MaxAge: 1000,
			},
			Validator: tmtypes.ValidatorParams{
				PubKeyTypes: []string{tmtypes.ABCIPubKeyTypeEd25519},
			},
		},
		AppHash:  appHash[:],
		AppState: json.RawMessage(appStateJSON),
	}

	err = genesis.ValidateAndComplete()
	if err != nil {
		return nil, err
	}

	return &genesis, nil
}

func MakeValidatorsAndCandidates(pubkeys []string, stake *big.Int) ([]types.Validator, []types.Candidate) {
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
