package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"log"
	"time"
)

var (
	height      = flag.Uint64("height", 0, "height")
	chainID     = flag.String("chain_id", "", "chain_id")
	genesisTime = flag.Duration("genesis_time", 0, "genesis_time")
)

func main() {
	required := []string{"height", "chain_id", "genesis_time"}
	flag.Parse()
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

	cdc := amino.NewCodec()

	jsonBytes, err := cdc.MarshalJSONIndent(currentState.Export11(*height), "", "	")
	if err != nil {
		panic(err)
	}

	appHash := [32]byte{}

	// Compose Genesis
	genesis := GenesisDoc{
		GenesisTime: time.Unix(0, 0).Add(*genesisTime),
		ChainID:     *chainID,
		ConsensusParams: &ConsensusParams{
			Block: BlockParams{
				MaxBytes:   10000000,
				MaxGas:     100000,
				TimeIotaMs: 1000,
			},
			Evidence: EvidenceParams{
				MaxAgeNumBlocks: 1000,
			},
			Validator: ValidatorParams{
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

	fmt.Println("OK")
	fmt.Println(sha3.Sum512([]byte(fmt.Sprintf("%v", genesis))))
}

type GenesisDoc struct {
	GenesisTime     time.Time          `json:"genesis_time"`
	ChainID         string             `json:"chain_id"`
	ConsensusParams *ConsensusParams   `json:"consensus_params,omitempty"`
	Validators      []GenesisValidator `json:"validators,omitempty"`
	AppHash         common.HexBytes    `json:"app_hash"`
	AppState        json.RawMessage    `json:"app_state,omitempty"`
}
type ConsensusParams struct {
	Block     BlockParams     `json:"block"`
	Evidence  EvidenceParams  `json:"evidence"`
	Validator ValidatorParams `json:"validator"`
}
type BlockParams struct {
	MaxBytes int64 `json:"max_bytes"`
	MaxGas   int64 `json:"max_gas"`
	// Minimum time increment between consecutive blocks (in milliseconds)
	// Not exposed to the application.
	TimeIotaMs int64 `json:"time_iota_ms"`
}
type GenesisValidator struct {
	Address Address       `json:"address"`
	PubKey  crypto.PubKey `json:"pub_key"`
	Power   int64         `json:"power"`
	Name    string        `json:"name"`
}
type Address = crypto.Address
type EvidenceParams struct {
	MaxAgeNumBlocks int64         `json:"max_age_num_blocks"` // only accept new evidence more recent than this
	MaxAgeDuration  time.Duration `json:"max_age_duration"`
}
type ValidatorParams struct {
	PubKeyTypes []string `json:"pub_key_types"`
}

const (
	MaxChainIDLen = 50
)

func (genDoc *GenesisDoc) ValidateAndComplete() error {
	if genDoc.ChainID == "" {
		return errors.New("genesis doc must include non-empty chain_id")
	}
	if len(genDoc.ChainID) > MaxChainIDLen {
		return errors.Errorf("chain_id in genesis doc is too long (max: %d)", MaxChainIDLen)
	}

	if genDoc.ConsensusParams == nil {
		genDoc.ConsensusParams = DefaultConsensusParams()
	} else if err := genDoc.ConsensusParams.Validate(); err != nil {
		return err
	}

	for i, v := range genDoc.Validators {
		if v.Power == 0 {
			return errors.Errorf("the genesis file cannot contain validators with no voting power: %v", v)
		}
		if len(v.Address) > 0 && !bytes.Equal(v.PubKey.Address(), v.Address) {
			return errors.Errorf("incorrect address for validator %v in the genesis file, should be %v", v, v.PubKey.Address())
		}
		if len(v.Address) == 0 {
			genDoc.Validators[i].Address = v.PubKey.Address()
		}
	}

	if genDoc.GenesisTime.IsZero() {
		genDoc.GenesisTime = tmtime.Now()
	}

	return nil
}
func DefaultConsensusParams() *ConsensusParams {
	return &ConsensusParams{
		DefaultBlockParams(),
		DefaultEvidenceParams(),
		DefaultValidatorParams(),
	}
}
func DefaultBlockParams() BlockParams {
	return BlockParams{
		MaxBytes:   22020096, // 21MB
		MaxGas:     -1,
		TimeIotaMs: 1000, // 1s
	}
}

// DefaultEvidenceParams Params returns a default EvidenceParams.
func DefaultEvidenceParams() EvidenceParams {
	return EvidenceParams{
		MaxAgeNumBlocks: 100000, // 27.8 hrs at 1block/s
		MaxAgeDuration:  48 * time.Hour,
	}
}

// DefaultValidatorParams returns a default ValidatorParams, which allows
// only ed25519 pubkeys.
func DefaultValidatorParams() ValidatorParams {
	return ValidatorParams{[]string{types.ABCIPubKeyTypeEd25519}}
}

const (
	ABCIPubKeyTypeEd25519   = "ed25519"
	ABCIPubKeyTypeSr25519   = "sr25519"
	ABCIPubKeyTypeSecp256k1 = "secp256k1"

	MaxBlockSizeBytes = 104857600
)

var ABCIPubKeyTypesToAminoNames = map[string]string{
	ABCIPubKeyTypeEd25519:   ed25519.PubKeyAminoName,
	ABCIPubKeyTypeSr25519:   "tendermint/PubKeySr25519",
	ABCIPubKeyTypeSecp256k1: secp256k1.PubKeyAminoName,
}

func (params *ConsensusParams) Validate() error {
	if params.Block.MaxBytes <= 0 {
		return errors.Errorf("block.MaxBytes must be greater than 0. Got %d",
			params.Block.MaxBytes)
	}
	if params.Block.MaxBytes > MaxBlockSizeBytes {
		return errors.Errorf("block.MaxBytes is too big. %d > %d",
			params.Block.MaxBytes, MaxBlockSizeBytes)
	}

	if params.Block.MaxGas < -1 {
		return errors.Errorf("block.MaxGas must be greater or equal to -1. Got %d",
			params.Block.MaxGas)
	}

	if params.Block.TimeIotaMs <= 0 {
		return errors.Errorf("block.TimeIotaMs must be greater than 0. Got %v",
			params.Block.TimeIotaMs)
	}

	if params.Evidence.MaxAgeNumBlocks <= 0 {
		return errors.Errorf("evidenceParams.MaxAgeNumBlocks must be greater than 0. Got %d",
			params.Evidence.MaxAgeNumBlocks)
	}

	if params.Evidence.MaxAgeDuration <= 0 {
		return errors.Errorf("evidenceParams.MaxAgeDuration must be grater than 0 if provided, Got %v",
			params.Evidence.MaxAgeDuration)
	}

	if len(params.Validator.PubKeyTypes) == 0 {
		return errors.New("len(Validator.PubKeyTypes) must be greater than 0")
	}

	// Check if keyType is a known ABCIPubKeyType
	for i := 0; i < len(params.Validator.PubKeyTypes); i++ {
		keyType := params.Validator.PubKeyTypes[i]
		if _, ok := ABCIPubKeyTypesToAminoNames[keyType]; !ok {
			return errors.Errorf("params.Validator.PubKeyTypes[%d], %s, is an unknown pubkey type",
				i, keyType)
		}
	}

	return nil
}
func (genDoc *GenesisDoc) SaveAs(file string) error {
	genDocBytes, err := types.GetCodec().MarshalJSONIndent(genDoc, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, genDocBytes, 0644)
}
