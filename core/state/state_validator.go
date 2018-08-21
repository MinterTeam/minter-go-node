// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

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
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateValidators struct {
	data Validators
	db   *StateDB

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	onDirty func() // Callback method to mark a state object newly dirty
}

type Validators []Validator

// empty returns whether the candidate is considered empty.
func (c *stateValidators) empty() bool {
	return false
}

func (c *stateValidators) Data() Validators {
	return c.data
}

type Validator struct {
	CandidateAddress types.Address
	TotalBipStake    *big.Int
	PubKey           types.Pubkey
	Commission       uint
	AccumReward      *big.Int
	AbsentTimes      uint

	tmAddress *[20]byte
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

func (c *stateValidators) deepCopy(db *StateDB, onDirty func()) *stateValidators {
	stateValidators := newValidator(db, c.data, onDirty)
	return stateValidators
}
