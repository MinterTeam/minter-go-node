package genesis

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/common"
	tmtypes "github.com/tendermint/tendermint/types"
	"math/big"
	"time"
)

func GetTestnetGenesis() *tmtypes.GenesisDoc {

	validatorPubKeyBytes, _ := base64.StdEncoding.DecodeString("qu4d3zD/VMkHFdkotWZS/FEb7Tci5Ylz6O+Ub12uOXk=")
	var validatorPubKey crypto.PubKeyEd25519
	copy(validatorPubKey[:], validatorPubKeyBytes)

	appHash, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")

	appState := AppState{
		FirstValidatorAddress: types.HexToAddress("Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f"),
		InitialBalances: []Account{
			{
				Address: types.HexToAddress("Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f"),
				Balance: map[string]string{
					"MNT": helpers.BipToPip(big.NewInt(1000000000)).String(),
				},
			},
			{
				Address: types.HexToAddress("Mxfe60014a6e9ac91618f5d1cab3fd58cded61ee99"),
				Balance: map[string]string{
					"MNT": helpers.BipToPip(big.NewInt(10000000)).String(),
				},
			},
		},
	}

	appStateJSON, _ := json.Marshal(appState)

	genesis := tmtypes.GenesisDoc{
		GenesisTime:     time.Date(2018, 7, 23, 0, 0, 0, 0, time.UTC),
		ChainID:         "minter-test-network-14",
		ConsensusParams: nil,
		Validators: []tmtypes.GenesisValidator{
			{
				PubKey: validatorPubKey,
				Power:  100,
			},
		},
		AppHash:  common.HexBytes(appHash),
		AppState: json.RawMessage([]byte(appStateJSON)),
	}

	genesis.ValidateAndComplete()

	return &genesis
}

type AppState struct {
	FirstValidatorAddress types.Address `json:"first_validator_address"`
	InitialBalances       []Account     `json:"initial_balances"`
}

type Account struct {
	Address types.Address     `json:"address"`
	Balance map[string]string `json:"balance"`
}
