package genesis

import (
	"encoding/base64"
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmtypes "github.com/tendermint/tendermint/types"
	"math/big"
	"time"
)

var (
	Network = "minter-test-network-22"
)

func GetTestnetGenesis() (*tmtypes.GenesisDoc, error) {
	validatorsPubKeys := []string{
		"SuHuc+YTbIWwypM6mhNHdYozSIXxCzI4OYpnrC6xU7g=",
		"c42kG6ant9abcpSvoVi4nFobQQy/DCRDyFxf4krR3Rw=",
		"bxbB/yGm+5RqrtD0wfzKJyty/ZBJiPkdOIMoK4rjG6I=",
		"nhPy9UaN14KzFkRPvWZZXhPbp9e9Pvob7NULQgRfWMY=",
	}
	validators := make([]tmtypes.GenesisValidator, len(validatorsPubKeys))

	for i, val := range validatorsPubKeys {
		validatorPubKeyBytes, _ := base64.StdEncoding.DecodeString(val)
		var validatorPubKey ed25519.PubKeyEd25519
		copy(validatorPubKey[:], validatorPubKeyBytes)

		validators[i] = tmtypes.GenesisValidator{
			PubKey: validatorPubKey,
			Power:  int64(100000000 / len(validatorsPubKeys)),
		}
	}

	appHash := [16]byte{}

	appState := AppState{
		FirstValidatorAddress: types.HexToAddress("Mxee81347211c72524338f9680072af90744333146"),
		InitialBalances: []Account{
			{
				Address: types.HexToAddress("Mxee81347211c72524338f9680072af90744333146"),
				Balance: map[string]string{
					"MNT": helpers.BipToPip(big.NewInt(100000000)).String(),
				},
			},
			{
				Address: types.HexToAddress("Mxfe60014a6e9ac91618f5d1cab3fd58cded61ee99"),
				Balance: map[string]string{
					"MNT": helpers.BipToPip(big.NewInt(100000000)).String(),
				},
			},
		},
	}

	appStateJSON, err := json.Marshal(appState)

	if err != nil {
		return nil, err
	}

	genesis := tmtypes.GenesisDoc{
		ChainID:         Network,
		GenesisTime:     time.Date(2018, 9, 19, 9, 0, 0, 0, time.UTC),
		ConsensusParams: nil,
		Validators:      validators,
		AppHash:         appHash[:],
		AppState:        json.RawMessage(appStateJSON),
	}

	err = genesis.ValidateAndComplete()

	if err != nil {
		return nil, err
	}

	return &genesis, nil
}

type AppState struct {
	FirstValidatorAddress types.Address `json:"first_validator_address"`
	InitialBalances       []Account     `json:"initial_balances"`
}

type Account struct {
	Address types.Address     `json:"address"`
	Balance map[string]string `json:"balance"`
}
