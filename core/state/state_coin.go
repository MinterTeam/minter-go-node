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
	"minter/crypto"
	"minter/rlp"
)

// stateCoin represents a coin which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateCoin struct {
	symbol     types.CoinSymbol
	symbolHash types.Hash
	data       Coin
	db         *StateDB

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	onDirty func(symbol types.CoinSymbol) // Callback method to mark a state object newly dirty
}

// empty returns whether the coin is considered empty.
func (c *stateCoin) empty() bool {
	return false
}

type Coin struct {
	Name           string
	Symbol         types.CoinSymbol
	Volume         *big.Int
	Crr            uint
	ReserveCoin    types.CoinSymbol
	ReserveBalance *big.Int
	Creator        types.Address
}

func (coin Coin) String() string {
	return fmt.Sprintf("%s (%s), volume: %s, reserve: %s, crr: %d", coin.Name, coin.Symbol, coin.Volume, coin.ReserveBalance, coin.Crr)
}

// newCoin creates a state object.
func newCoin(db *StateDB, symbol types.CoinSymbol, data Coin, onDirty func(symbol types.CoinSymbol)) *stateCoin {
	coin := &stateCoin{
		db:         db,
		symbol:     symbol,
		symbolHash: crypto.Keccak256Hash(symbol[:]),
		data:       data,
		onDirty:    onDirty,
	}

	coin.onDirty(coin.symbol)

	return coin
}

// EncodeRLP implements rlp.Encoder.
func (c *stateCoin) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (c *stateCoin) deepCopy(db *StateDB, onDirty func(addr types.CoinSymbol)) *stateCoin {
	stateCoin := newCoin(db, c.symbol, c.data, onDirty)
	return stateCoin
}

func (c *stateCoin) AddVolume(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetVolume(new(big.Int).Add(c.Volume(), amount))
}

func (c *stateCoin) SubVolume(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetVolume(new(big.Int).Sub(c.Volume(), amount))
}

func (c *stateCoin) SetVolume(amount *big.Int) {
	c.setVolume(amount)
}

func (c *stateCoin) setVolume(amount *big.Int) {

	c.data.Volume = amount

	if c.onDirty != nil {
		c.onDirty(c.Symbol())
		c.onDirty = nil
	}
}

func (c *stateCoin) AddReserve(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetReserve(new(big.Int).Add(c.ReserveBalance(), amount))
}

func (c *stateCoin) SubReserve(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetReserve(new(big.Int).Sub(c.ReserveBalance(), amount))
}

func (c *stateCoin) SetReserve(amount *big.Int) {
	c.setReserve(amount)
}

func (c *stateCoin) setReserve(amount *big.Int) {

	c.data.ReserveBalance = amount

	if c.onDirty != nil {
		c.onDirty(c.Symbol())
		c.onDirty = nil
	}
}

//
// Attribute accessors
//

func (c *stateCoin) Symbol() types.CoinSymbol {
	return c.symbol
}

func (c *stateCoin) Data() Coin {
	return c.data
}

func (c *stateCoin) Volume() *big.Int {
	return c.data.Volume
}

func (c *stateCoin) ReserveBalance() *big.Int {
	return c.data.ReserveBalance
}
