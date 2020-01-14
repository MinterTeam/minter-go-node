package checks

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/check"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/tree"
	"sort"
	"sync"
)

const mainPrefix = byte('t')

type Checks struct {
	usedChecks map[types.Hash]struct{}

	iavl tree.Tree

	lock sync.RWMutex
}

func NewChecks(iavl tree.Tree) (*Checks, error) {
	return &Checks{iavl: iavl, usedChecks: map[types.Hash]struct{}{}}, nil
}

func (c *Checks) Commit() error {
	for _, hash := range c.getOrderedHashes() {
		c.lock.Lock()
		delete(c.usedChecks, hash)
		c.lock.Unlock()

		trieHash := append([]byte{mainPrefix}, hash.Bytes()...)
		c.iavl.Set(trieHash, []byte{0x1})
	}

	return nil
}

func (c *Checks) IsCheckUsed(check *check.Check) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if _, has := c.usedChecks[check.Hash()]; has {
		return true
	}

	_, data := c.iavl.Get(append([]byte{mainPrefix}, check.Hash().Bytes()...))

	return len(data) != 0
}

func (c *Checks) UseCheck(check *check.Check) {
	c.UseCheckHash(check.Hash())
}

func (c *Checks) UseCheckHash(hash types.Hash) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.usedChecks[hash] = struct{}{}
}

func (c *Checks) Export(state *types.AppState) {
	// todo: iterate range?
	c.iavl.Iterate(func(key []byte, value []byte) bool {
		if key[0] == mainPrefix {
			state.UsedChecks = append(state.UsedChecks, types.UsedCheck(fmt.Sprintf("%x", key[1:])))
		}

		return false
	})
}

func (c *Checks) getOrderedHashes() []types.Hash {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var keys []types.Hash
	for hash := range c.usedChecks {
		keys = append(keys, hash)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}
