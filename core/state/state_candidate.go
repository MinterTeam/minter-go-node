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

	"minter/rlp"
	"minter/core/types"
	"fmt"
	"math/big"
	"bytes"
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
	Value *big.Int
}

type Candidate struct {
	CandidateAddress       types.Address
	TotalStake             *big.Int
	PubKey                 types.Pubkey
	Commission             uint
	AccumReward            *big.Int
	Stakes                 []Stake
	CreatedAtBlock         uint
	Status                 byte
	AbsentTimes            uint
}

func (candidate Candidate) GetStakeOfAddress(addr types.Address) *Stake {
	for i, stake := range candidate.Stakes {
		if bytes.Compare(stake.Owner.Bytes(), addr.Bytes()) == 0 {
			return &(candidate.Stakes[i])
		}
	}

	return nil
}

func (candidate Candidate) String() string {
	return fmt.Sprintf("Candidate")
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
