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
	"math/big"
	"minter/core/types"
	"minter/rlp"
	"bytes"
)

// stateFrozenFund represents a frozen fund which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateFrozenFund struct {
	blockHeight int64
	deleted     bool
	data        FrozenFunds
	db          *StateDB

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	onDirty func(blockHeight int64) // Callback method to mark a state object newly dirty
}

// empty returns whether the coin is considered empty.
func (c *stateFrozenFund) empty() bool {
	return false
}

// frozen funds are only for BaseCoin
type FrozenFund struct {
	Address      types.Address
	CandidateKey []byte
	Value        *big.Int
}

type FrozenFunds struct {
	BlockHeight int64
	List        []FrozenFund
}

func (f FrozenFunds) String() string {
	return fmt.Sprintf("Frozen funds at block %d (%d items)", f.BlockHeight, len(f.List))
}

// newFrozenFund creates a state object.
func newFrozenFund(db *StateDB, blockHeight int64, data FrozenFunds, onDirty func(blockHeight int64)) *stateFrozenFund {
	frozenFund := &stateFrozenFund{
		db:          db,
		blockHeight: blockHeight,
		data:        data,
		onDirty:     onDirty,
	}

	frozenFund.onDirty(frozenFund.blockHeight)

	return frozenFund
}

// EncodeRLP implements rlp.Encoder.
func (c *stateFrozenFund) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (c *stateFrozenFund) deepCopy(db *StateDB, onDirty func(blockHeight int64)) *stateFrozenFund {
	frozenFund := newFrozenFund(db, c.blockHeight, c.data, onDirty)
	return frozenFund
}

func (c *stateFrozenFund) Delete() {
	c.deleted = true
}

func (c *stateFrozenFund) AddFund(address types.Address, candidateKey []byte, value *big.Int) {
	c.addFund(FrozenFund{
		Address:      address,
		CandidateKey: candidateKey,
		Value:        value,
	})
}

func (c *stateFrozenFund) addFund(fund FrozenFund) {

	c.data.List = append(c.data.List, fund)

	if c.onDirty != nil {
		c.onDirty(c.blockHeight)
		c.onDirty = nil
	}
}

// remove fund with given candidate key (used in byzantine validator's punishment)
func (c *stateFrozenFund) RemoveFund(candidateKey []byte) {
	c.removeFund(candidateKey)
}

func (c *stateFrozenFund) removeFund(candidateKey []byte) {
	var NewList []FrozenFund

	for _, item := range c.data.List {
		// skip fund with given candidate key
		if bytes.Compare(item.CandidateKey, candidateKey) == 0 {
			continue
		}

		NewList = append(NewList, item)
	}

	c.data.List = NewList

	if c.onDirty != nil {
		c.onDirty(c.blockHeight)
		c.onDirty = nil
	}
}

//
// Attribute accessors
//

func (c *stateFrozenFund) BlockHeight() int64 {
	return c.blockHeight
}

func (c *stateFrozenFund) List() []FrozenFund {
	return c.data.List
}

func (c *stateFrozenFund) Data() FrozenFunds {
	return c.data
}
