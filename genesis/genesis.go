package genesis

import (
	"encoding/base64"
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/developers"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmtypes "github.com/tendermint/tendermint/types"
	"math/big"
	"time"
)

var (
	Network     = "minter-test-network-33"
	genesisTime = time.Date(2019, 2, 18, 9, 0, 0, 0, time.UTC)

	totalValidatorsPower = 100000000
)

type AppState struct {
	FirstValidatorAddress types.Address `json:"first_validator_address"`
	InitialBalances       []Account     `json:"initial_balances"`
}

type Account struct {
	Address types.Address     `json:"address"`
	Balance map[string]string `json:"balance"`
}

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

	// Prepare initial AppState
	appStateJSON, err := json.Marshal(AppState{
		FirstValidatorAddress: developers.Address,
		InitialBalances:       makeBalances(balances),
	})
	if err != nil {
		return nil, err
	}

	appHash := [32]byte{}

	// Compose Genesis
	genesis := tmtypes.GenesisDoc{
		ChainID:     Network,
		GenesisTime: genesisTime,
		Validators:  makeValidators(validatorsPubKeys),
		AppHash:     appHash[:],
		AppState:    json.RawMessage(appStateJSON),
	}

	err = genesis.ValidateAndComplete()
	if err != nil {
		return nil, err
	}

	return &genesis, nil
}

func decodeValidatorPubkey(pubkey string) ed25519.PubKeyEd25519 {
	validatorPubKeyBytes, err := base64.StdEncoding.DecodeString(pubkey)
	if err != nil {
		panic(err)
	}

	var validatorPubKey ed25519.PubKeyEd25519
	copy(validatorPubKey[:], validatorPubKeyBytes)

	return validatorPubKey
}

func makeValidators(pubkeys []string) []tmtypes.GenesisValidator {
	validators := make([]tmtypes.GenesisValidator, len(pubkeys))
	for i, val := range pubkeys {
		validators[i] = tmtypes.GenesisValidator{
			PubKey: decodeValidatorPubkey(val),
			Power:  int64(totalValidatorsPower / len(pubkeys)),
		}
	}

	return validators
}

func makeBalances(balances map[string]int64) []Account {
	var totalBalances int64
	for _, val := range balances {
		totalBalances += val
	}

	balances[developers.Address.String()] = 200000000 - totalBalances // Developers account

	result := make([]Account, len(balances))
	i := 0
	for address, balance := range balances {
		result[i] = Account{
			Address: types.HexToAddress(address),
			Balance: map[string]string{
				types.GetBaseCoin().String(): helpers.BipToPip(big.NewInt(balance)).String(),
			},
		}
		i++
	}

	return result
}
