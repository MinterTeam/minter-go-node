package halts

import (
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"sort"
	"sync"
)

const mainPrefix = byte('h')

type HaltBlocks struct {
	list  map[uint64]*Model
	dirty map[uint64]interface{}

	bus  *bus.Bus
	iavl tree.Tree

	lock sync.RWMutex
}

func NewHalts(stateBus *bus.Bus, iavl tree.Tree) (*HaltBlocks, error) {
	halts := &HaltBlocks{
		bus:   stateBus,
		iavl:  iavl,
		list:  map[uint64]*Model{},
		dirty: map[uint64]interface{}{},
	}

	halts.bus.SetHaltBlocks(NewBus(halts))

	return halts, nil
}

func (f *HaltBlocks) Commit() error {
	dirty := f.getOrderedDirty()
	for _, height := range dirty {
		ff := f.getFromMap(height)

		f.lock.Lock()
		delete(f.dirty, height)
		f.lock.Unlock()

		data, err := rlp.EncodeToBytes(ff)
		if err != nil {
			return fmt.Errorf("can't encode object at %d: %v", height, err)
		}

		path := getPath(height)
		f.iavl.Set(path, data)
	}

	return nil
}

func (f *HaltBlocks) GetHaltBlocks(height uint64) *Model {
	return f.get(height)
}

func (f *HaltBlocks) GetOrNew(height uint64) *Model {
	ff := f.get(height)
	if ff == nil {
		ff = &Model{
			height:    height,
			markDirty: f.markDirty,
		}
		f.setToMap(height, ff)
	}

	return ff
}

func (f *HaltBlocks) get(height uint64) *Model {
	if ff := f.getFromMap(height); ff != nil {
		return ff
	}

	_, enc := f.iavl.Get(getPath(height))
	if len(enc) == 0 {
		return nil
	}

	ff := &Model{}
	if err := rlp.DecodeBytes(enc, ff); err != nil {
		panic(fmt.Sprintf("failed to decode halt blocks at height %d: %s", height, err))
	}

	ff.height = height
	ff.markDirty = f.markDirty

	f.setToMap(height, ff)

	return ff
}

func (f *HaltBlocks) markDirty(height uint64) {
	f.dirty[height] = struct{}{}
}

func (f *HaltBlocks) getOrderedDirty() []uint64 {
	keys := make([]uint64, 0, len(f.dirty))
	for k := range f.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func (f *HaltBlocks) AddHaltBlock(height uint64, pubkey types.Pubkey) {
	f.GetOrNew(height).addHaltBlock(pubkey)
}

func (f *HaltBlocks) Delete(height uint64) {
	ff := f.get(height)
	if ff == nil {
		return
	}

	ff.delete()
}

func (f *HaltBlocks) Export(state *types.AppState, height uint64) {
	for i := height; i <= height; i++ {
		halts := f.get(i)
		if halts == nil {
			continue
		}

		for _, haltBlock := range halts.List {
			state.HaltBlocks = append(state.HaltBlocks, types.HaltBlock{
				Height:       i,
				CandidateKey: haltBlock.CandidateKey,
			})
		}
	}
}

func (f *HaltBlocks) getFromMap(height uint64) *Model {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.list[height]
}

func (f *HaltBlocks) setToMap(height uint64, model *Model) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.list[height] = model
}

func getPath(height uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, height)

	return append([]byte{mainPrefix}, b...)
}
