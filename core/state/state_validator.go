package state

import (
	"io"

	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
)

// stateValidators represents a validators which is being modified.
type stateValidators struct {
	data Validators
	db   *StateDB

	onDirty func() // Callback method to mark a state object newly dirty
}

type Validators []Validator

func (c *stateValidators) Data() Validators {
	return c.data
}

type Validator struct {
	RewardAddress types.Address
	TotalBipStake *big.Int
	PubKey        types.Pubkey
	Commission    uint
	AccumReward   *big.Int
	AbsentTimes   *types.BitArray

	tmAddress *[20]byte
	toDrop    bool
}

func (validator *Validator) CountAbsentTimes() int {
	count := 0

	for i := 0; i < ValidatorMaxAbsentWindow; i++ {
		if validator.AbsentTimes.GetIndex(i) {
			count++
		}
	}

	return count
}

func (validator *Validator) IsToDrop() bool {
	return validator.toDrop
}

func (validator Validator) GetAddress() [20]byte {

	if validator.tmAddress != nil {
		return *validator.tmAddress
	}

	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], validator.PubKey)

	var address [20]byte
	copy(address[:], pubkey.Address().Bytes())

	validator.tmAddress = &address

	return address
}

func (validator Validator) String() string {
	return fmt.Sprintf("Validator")
}

// newValidator creates a state object.
func newValidator(db *StateDB, data Validators, onDirty func()) *stateValidators {
	validators := &stateValidators{
		db:      db,
		data:    data,
		onDirty: onDirty,
	}

	validators.onDirty()

	return validators
}

// EncodeRLP implements rlp.Encoder.
func (c *stateValidators) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (c *stateValidators) SetData(validators Validators) {
	c.data = validators
}
