package checks

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/MinterTeam/minter-go-node/coreV2/check"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/cosmos/iavl"
)

const mainPrefix = byte('t')

type RChecks interface {
	Export(state *types.AppState)
	IsCheckUsed(check *check.Check) bool
}

type Checks struct {
	usedChecks map[types.Hash]struct{}

	db atomic.Value

	lock sync.RWMutex
}

func NewChecks(db *iavl.ImmutableTree) *Checks {
	immutableTree := atomic.Value{}
	if db != nil {
		immutableTree.Store(db)
	}
	return &Checks{db: immutableTree, usedChecks: map[types.Hash]struct{}{}}
}

func (c *Checks) immutableTree() *iavl.ImmutableTree {
	db := c.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (c *Checks) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	c.db.Store(immutableTree)
}

func (c *Checks) Commit(db *iavl.MutableTree, version int64) error {
	hashes := c.getOrderedHashes()
	for _, hash := range hashes {
		c.lock.Lock()
		delete(c.usedChecks, hash)
		c.lock.Unlock()

		trieHash := append([]byte{mainPrefix}, hash.Bytes()...)
		db.Set(trieHash, []byte{0x1})
	}

	return nil
}

func (c *Checks) IsCheckUsed(check *check.Check) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if _, has := c.usedChecks[check.Hash()]; has {
		return true
	}

	_, data := c.immutableTree().Get(append([]byte{mainPrefix}, check.Hash().Bytes()...))

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
	c.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		state.UsedChecks = append(state.UsedChecks, types.UsedCheck(fmt.Sprintf("%x", key[1:])))
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
