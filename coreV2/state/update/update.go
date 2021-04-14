package update

import (
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
	"sort"
	"sync"
	"sync/atomic"
)

const mainPrefix = byte('u')

type RUpdate interface {
	// Deprecated
	ExportV1(state *types.AppState, id types.CoinID)

	Export(state *types.AppState)
	GetVotes(height uint64) []*Model
	IsVoteExists(height uint64, pubkey types.Pubkey) bool
}

type Update struct {
	list      map[uint64][]*Model
	dirty     map[uint64]struct{}
	forDelete uint64

	db   atomic.Value
	lock sync.RWMutex
}

func New(db *iavl.ImmutableTree) *Update {
	immutableTree := atomic.Value{}
	if db != nil {
		immutableTree.Store(db)
	}
	halts := &Update{
		db:        immutableTree,
		list:      map[uint64][]*Model{},
		dirty:     map[uint64]struct{}{},
		forDelete: 0,
	}

	return halts
}

func (c *Update) immutableTree() *iavl.ImmutableTree {
	db := c.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (c *Update) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	c.db.Store(immutableTree)
}

func (c *Update) Export(state *types.AppState) {
	c.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		if len(key) < 8 {
			return false
		}
		height := binary.LittleEndian.Uint64(key[1:])
		updates := c.get(height)
		if updates == nil {
			return false
		}

		for _, u := range updates {
			state.UpdateVotes = append(state.UpdateVotes, types.UpdateVote{
				Height:  height,
				Votes:   u.Votes,
				Version: u.Version,
			})
		}

		return false
	})

}

// Deprecated
func (c *Update) ExportV1(state *types.AppState, id types.CoinID) {
	return
}

func (c *Update) Commit(db *iavl.MutableTree) error {
	c.lock.RLock()
	dirties := c.getOrderedDirty()
	c.lock.RUnlock()
	for _, height := range dirties {
		models := c.getFromMap(height)

		c.lock.Lock()
		delete(c.dirty, height)
		c.lock.Unlock()

		data, err := rlp.EncodeToBytes(models)
		if err != nil {
			return fmt.Errorf("can't encode object at %d: %v", height, err)
		}

		db.Set(getPath(height), data)
	}

	if c.forDelete != 0 {
		path := getPath(c.forDelete)
		db.Remove(path)
		c.lock.Lock()
		delete(c.list, c.forDelete)
		c.forDelete = 0
		c.lock.Unlock()
	}

	return nil
}

func (c *Update) GetVotes(height uint64) []*Model {
	return c.get(height)
}

func (c *Update) getOrNew(height uint64, version string) *Model {
	models := c.get(height)

	for _, model := range models {
		if version == model.Version {
			return model
		}
	}

	price := &Model{
		height:    height,
		Version:   version,
		markDirty: c.markDirty(height),
	}
	c.setToMap(height, append(models, price))
	return price
}

func (c *Update) get(height uint64) []*Model {
	if haltBlock := c.getFromMap(height); haltBlock != nil {
		return haltBlock
	}

	_, enc := c.immutableTree().Get(getPath(height))
	if len(enc) == 0 {
		return nil
	}

	var voteBlock []*Model
	if err := rlp.DecodeBytes(enc, &voteBlock); err != nil {
		panic(fmt.Sprintf("failed to decode halt blocks at height %d: %s", height, err))
	}

	c.setToMap(height, voteBlock)

	return voteBlock
}

func (c *Update) markDirty(height uint64) func() {
	return func() {
		c.lock.Lock()
		defer c.lock.Unlock()
		c.dirty[height] = struct{}{}
	}
}

func (c *Update) getOrderedDirty() []uint64 {
	keys := make([]uint64, 0, len(c.dirty))
	for k := range c.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func (c *Update) IsVoteExists(height uint64, pubkey types.Pubkey) bool {
	model := c.get(height)
	if len(model) == 0 {
		return false
	}

	for _, price := range model {
		for _, vote := range price.Votes {
			if vote == pubkey {
				return true
			}
		}
	}

	return false
}

func (c *Update) AddVote(height uint64, pubkey types.Pubkey, version string) {
	c.getOrNew(height, version).addVote(pubkey)
}

func (c *Update) Delete(height uint64) {
	prices := c.get(height)
	if len(prices) == 0 {
		return
	}

	c.lock.RLock()
	defer c.lock.RUnlock()

	c.forDelete = height
}

func (c *Update) getFromMap(height uint64) []*Model {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.list[height]
}

func (c *Update) setToMap(height uint64, model []*Model) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.list[height] = model
}

func getPath(height uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, height)

	return append([]byte{mainPrefix}, b...)
}
