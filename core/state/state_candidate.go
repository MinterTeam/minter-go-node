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

	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
)

const (
	CandidateStatusOffline = 0x01
	CandidateStatusOnline  = 0x02
)

// stateCandidate represents a candidate which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateCandidates struct {
	data Candidates
	db   *StateDB

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	onDirty func() // Callback method to mark a state object newly dirty
}

type Candidates []Candidate

// empty returns whether the candidate is considered empty.
func (c *stateCandidates) empty() bool {
	return false
}

type Stake struct {
	Owner types.Address
	Coin  types.CoinSymbol
	Value *big.Int
}

func (s *Stake) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Owner types.Address    `json:"owner"`
		Coin  types.CoinSymbol `json:"coin"`
		Value string           `json:"value"`
	}{
		Owner: s.Owner,
		Coin:  s.Coin,
		Value: s.Value.String(),
	})
}

func (s *Stake) BipValue(context *StateDB) *big.Int {
	if s.Coin.IsBaseCoin() {
		return big.NewInt(0).Set(s.Value)
	}

	if _, has := context.stakeCache[s.Coin]; !has {
		totalStaked := big.NewInt(0)
		candidates := context.getStateCandidates()

		for _, candidate := range candidates.data {
			for _, stake := range candidate.Stakes {
				if bytes.Equal(stake.Coin.Bytes(), s.Coin.Bytes()) {
					totalStaked.Add(totalStaked, stake.Value)
				}
			}
		}

		coin := context.getStateCoin(s.Coin)
		context.stakeCache[s.Coin] = StakeCache{
			TotalValue: totalStaked,
			BipValue:   formula.CalculateSaleReturn(coin.Volume(), coin.ReserveBalance(), coin.data.Crr, totalStaked),
		}
	}

	data := context.stakeCache[s.Coin]

	value := big.NewInt(0).Set(data.BipValue)
	value.Mul(value, s.Value)
	value.Div(value, data.TotalValue)

	return value
}

type Candidate struct {
	CandidateAddress types.Address
	TotalBipStake    *big.Int
	PubKey           types.Pubkey
	Commission       uint
	Stakes           []Stake
	CreatedAtBlock   uint
	Status           byte

	tmAddress *[20]byte
}

func (candidate Candidate) GetStakeOfAddress(addr types.Address, coin types.CoinSymbol) *Stake {
	for i, stake := range candidate.Stakes {
		if bytes.Equal(stake.Coin.Bytes(), coin.Bytes()) && bytes.Equal(stake.Owner.Bytes(), addr.Bytes()) {
			return &(candidate.Stakes[i])
		}
	}

	return nil
}

func (candidate Candidate) String() string {
	return fmt.Sprintf("Candidate")
}

func (candidate Candidate) GetAddress() [20]byte {
	if candidate.tmAddress != nil {
		return *candidate.tmAddress
	}

	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], candidate.PubKey)

	var address [20]byte
	copy(address[:], pubkey.Address().Bytes())

	candidate.tmAddress = &address

	return address
}

// newCandidate creates a state object.
func newCandidate(db *StateDB, data Candidates, onDirty func()) *stateCandidates {
	candidate := &stateCandidates{
		db:      db,
		data:    data,
		onDirty: onDirty,
	}

	candidate.onDirty()

	return candidate
}

// EncodeRLP implements rlp.Encoder.
func (c *stateCandidates) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (c *stateCandidates) deepCopy(db *StateDB, onDirty func()) *stateCandidates {
	stateCandidate := newCandidate(db, c.data, onDirty)
	return stateCandidate
}

func (c *stateCandidates) GetData() Candidates {
	return c.data
}
