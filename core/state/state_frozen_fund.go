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
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"io"

	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
)

// stateFrozenFund represents a frozen fund which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateFrozenFund struct {
	blockHeight uint64
	deleted     bool
	data        FrozenFunds
	db          *StateDB

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	onDirty func(blockHeight uint64) // Callback method to mark a state object newly dirty
}

// empty returns whether the coin is considered empty.
func (c *stateFrozenFund) empty() bool {
	return false
}

// frozen funds are only for BaseCoin
type FrozenFund struct {
	Address      types.Address
	CandidateKey []byte
	Coin         types.CoinSymbol
	Value        *big.Int
}

type FrozenFunds struct {
	BlockHeight uint64
	List        []FrozenFund
}

func (f FrozenFunds) String() string {
	return fmt.Sprintf("Frozen funds at block %d (%d items)", f.BlockHeight, len(f.List))
}

// newFrozenFund creates a state object.
func newFrozenFund(db *StateDB, blockHeight uint64, data FrozenFunds, onDirty func(blockHeight uint64)) *stateFrozenFund {
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

func (c *stateFrozenFund) deepCopy(db *StateDB, onDirty func(blockHeight uint64)) *stateFrozenFund {
	frozenFund := newFrozenFund(db, c.blockHeight, c.data, onDirty)
	return frozenFund
}

func (c *stateFrozenFund) Delete() {
	c.deleted = true
	c.onDirty(c.blockHeight)
}

func (c *stateFrozenFund) AddFund(address types.Address, candidateKey []byte, coin types.CoinSymbol, value *big.Int) {
	c.addFund(FrozenFund{
		Address:      address,
		CandidateKey: candidateKey,
		Coin:         coin,
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

// punish fund with given candidate key (used in byzantine validator's punishment)
func (c *stateFrozenFund) PunishFund(candidateAddress [20]byte) {
	c.punishFund(candidateAddress)
}

func (c *stateFrozenFund) punishFund(candidateAddress [20]byte) {

	edb := eventsdb.GetCurrent()

	var NewList []FrozenFund

	for _, item := range c.data.List {
		// skip fund with given candidate key
		var pubkey ed25519.PubKeyEd25519
		copy(pubkey[:], item.CandidateKey)

		var address [20]byte
		copy(address[:], pubkey.Address().Bytes())

		if candidateAddress == address {
			newValue := big.NewInt(0).Set(item.Value)
			newValue.Mul(newValue, big.NewInt(95))
			newValue.Div(newValue, big.NewInt(100))

			slashed := big.NewInt(0).Set(item.Value)
			slashed.Sub(slashed, newValue)

			edb.AddEvent(int64(c.blockHeight), eventsdb.SlashEvent{
				Address:         item.Address,
				Amount:          slashed.Bytes(),
				Coin:            item.Coin,
				ValidatorPubKey: item.CandidateKey,
			})

			item.Value = newValue
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

func (c *stateFrozenFund) BlockHeight() uint64 {
	return c.blockHeight
}

func (c *stateFrozenFund) List() []FrozenFund {
	return c.data.List
}

func (c *stateFrozenFund) Data() FrozenFunds {
	return c.data
}
