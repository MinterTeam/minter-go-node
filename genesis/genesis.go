package genesis

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/common"
	tmtypes "github.com/tendermint/tendermint/types"
	"math/big"
	"time"
)

func GetTestnetGenesis() (*tmtypes.GenesisDoc, error) {

	validatorPubKeyBytes, err := base64.StdEncoding.DecodeString("SuHuc+YTbIWwypM6mhNHdYozSIXxCzI4OYpnrC6xU7g=")

	if err != nil {
		return nil, err
	}

	var validatorPubKey ed25519.PubKeyEd25519
	copy(validatorPubKey[:], validatorPubKeyBytes)

	appHash, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")

	if err != nil {
		return nil, err
	}

	appState := AppState{
		FirstValidatorAddress: types.HexToAddress("Mxee81347211c72524338f9680072af90744333146"),
		InitialBalances: []Account{
			{
				Address: types.HexToAddress("Mxee81347211c72524338f9680072af90744333146"),
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

	appStateJSON, err := json.Marshal(appState)

	if err != nil {
		return nil, err
	}

	genesis := tmtypes.GenesisDoc{
		GenesisTime:     time.Date(2018, 7, 31, 0, 0, 0, 0, time.UTC),
		ChainID:         "minter-test-network-18",
		ConsensusParams: nil,
		Validators: []tmtypes.GenesisValidator{
			{
				PubKey: validatorPubKey,
				Power:  100000000,
			},
		},
		AppHash:  common.HexBytes(appHash),
		AppState: json.RawMessage([]byte(appStateJSON)),
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
