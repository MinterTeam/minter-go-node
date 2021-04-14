package halts

import (
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
	"sort"
	"sync"
	"sync/atomic"
)

const mainPrefix = byte('h')

type RHalts interface {
	Export(state *types.AppState)
	GetHaltBlocks(height uint64) *Model
	IsHaltExists(height uint64, pubkey types.Pubkey) bool
}

type HaltBlocks struct {
	list  map[uint64]*Model
	dirty map[uint64]struct{}

	bus *bus.Bus
	db  atomic.Value

	lock sync.RWMutex
}

func NewHalts(stateBus *bus.Bus, db *iavl.ImmutableTree) *HaltBlocks {
	immutableTree := atomic.Value{}
	if db != nil {
		immutableTree.Store(db)
	}
	halts := &HaltBlocks{
		bus:   stateBus,
		db:    immutableTree,
		list:  map[uint64]*Model{},
		dirty: map[uint64]struct{}{},
	}

	halts.bus.SetHaltBlocks(NewBus(halts))

	return halts
}

func (hb *HaltBlocks) immutableTree() *iavl.ImmutableTree {
	db := hb.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (hb *HaltBlocks) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	hb.db.Store(immutableTree)
}

func (hb *HaltBlocks) Commit(db *iavl.MutableTree) error {
	dirty := hb.getOrderedDirty()
	for _, height := range dirty {
		haltBlock := hb.get(height)
		path := getPath(height)

		hb.lock.Lock()
		delete(hb.dirty, height)
		if haltBlock.deleted {
			delete(hb.list, height)
			db.Remove(path)
		} else {
			data, err := rlp.EncodeToBytes(haltBlock)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", height, err)
			}

			db.Set(path, data)
		}
		hb.lock.Unlock()
	}

	return nil
}

func (hb *HaltBlocks) GetHaltBlocks(height uint64) *Model {
	return hb.get(height)
}

func (hb *HaltBlocks) GetOrNew(height uint64) *Model {
	haltBlock := hb.get(height)
	if haltBlock == nil {
		haltBlock = &Model{
			height:    height,
			markDirty: hb.markDirty,
		}
		hb.setToMap(height, haltBlock)
	}

	return haltBlock
}

func (hb *HaltBlocks) get(height uint64) *Model {
	if haltBlock := hb.getFromMap(height); haltBlock != nil {
		return haltBlock
	}

	_, enc := hb.immutableTree().Get(getPath(height))
	if len(enc) == 0 {
		return nil
	}

	haltBlock := &Model{}
	if err := rlp.DecodeBytes(enc, haltBlock); err != nil {
		panic(fmt.Sprintf("failed to decode halt blocks at height %d: %s", height, err))
	}

	haltBlock.height = height
	haltBlock.markDirty = hb.markDirty

	hb.setToMap(height, haltBlock)

	return haltBlock
}

func (hb *HaltBlocks) markDirty(height uint64) {
	hb.lock.Lock()
	defer hb.lock.Unlock()

	hb.dirty[height] = struct{}{}
}

func (hb *HaltBlocks) getOrderedDirty() []uint64 {
	hb.lock.RLock()
	keys := make([]uint64, 0, len(hb.dirty))
	for k := range hb.dirty {
		keys = append(keys, k)
	}
	hb.lock.RUnlock()

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func (hb *HaltBlocks) IsHaltExists(height uint64, pubkey types.Pubkey) bool {
	model := hb.get(height)
	if model == nil {
		return false
	}

	model.lock.Lock()
	defer model.lock.Unlock()

	for _, halt := range model.List {
		if halt.Pubkey == pubkey {
			return true
		}
	}

	return false
}

func (hb *HaltBlocks) AddHaltBlock(height uint64, pubkey types.Pubkey) {
	hb.GetOrNew(height).addHaltBlock(pubkey)
}

func (hb *HaltBlocks) Delete(height uint64) {
	haltBlock := hb.get(height)
	if haltBlock == nil {
		return
	}

	haltBlock.delete()
}

func (hb *HaltBlocks) Export(state *types.AppState) {
	hb.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		if len(key) < 8 {
			return false
		}

		height := binary.LittleEndian.Uint64(key[1:])
		halts := hb.get(height)
		if halts == nil {
			return false
		}

		for _, haltBlock := range halts.List {
			state.HaltBlocks = append(state.HaltBlocks, types.HaltBlock{
				Height:       height,
				CandidateKey: haltBlock.Pubkey,
			})
		}

		return false
	})
}

func (hb *HaltBlocks) getFromMap(height uint64) *Model {
	hb.lock.RLock()
	defer hb.lock.RUnlock()

	return hb.list[height]
}

func (hb *HaltBlocks) setToMap(height uint64, model *Model) {
	hb.lock.Lock()
	defer hb.lock.Unlock()

	hb.list[height] = model
}

func getPath(height uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, height)

	return append([]byte{mainPrefix}, b...)
}
