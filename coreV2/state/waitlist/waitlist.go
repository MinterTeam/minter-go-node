package waitlist

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
	"log"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
)

const mainPrefix = byte('w')

type RWaitList interface {
	// Deprecated
	ExportV1(state *types.AppState, ds []uint32)

	Get(address types.Address, pubkey types.Pubkey, coin types.CoinID) *Item
	GetByAddress(address types.Address) *Model
	GetByAddressAndPubKey(address types.Address, pubkey types.Pubkey) []Item
	Export(state *types.AppState)
}

type WaitList struct {
	list  map[types.Address]*Model
	dirty map[types.Address]struct{}

	db atomic.Value

	bus *bus.Bus

	lock sync.RWMutex
}

func NewWaitList(stateBus *bus.Bus, db *iavl.ImmutableTree) *WaitList {
	immutableTree := atomic.Value{}
	if db != nil {
		immutableTree.Store(db)
	}
	waitlist := &WaitList{
		bus:   stateBus,
		db:    immutableTree,
		list:  map[types.Address]*Model{},
		dirty: map[types.Address]struct{}{},
	}
	waitlist.bus.SetWaitList(NewBus(waitlist))

	return waitlist
}

func (wl *WaitList) immutableTree() *iavl.ImmutableTree {
	db := wl.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (wl *WaitList) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	wl.db.Store(immutableTree)
}

func (wl *WaitList) Export(state *types.AppState) {
	wl.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		address := types.BytesToAddress(key[1:])

		model := wl.GetByAddress(address)
		if model != nil && len(model.List) != 0 {
			for _, w := range model.List {
				state.Waitlist = append(state.Waitlist, types.Waitlist{
					CandidateID: uint64(w.CandidateId),
					Owner:       address,
					Coin:        uint64(w.Coin),
					Value:       w.Value.String(),
				})
			}
		}

		return false
	})

	sort.SliceStable(state.Waitlist, func(i, j int) bool {
		return bytes.Compare(state.Waitlist[i].Owner.Bytes(), state.Waitlist[j].Owner.Bytes()) == 1
	})
}

// Deprecated
func (wl *WaitList) ExportV1(state *types.AppState, droppedIDs []uint32) {
	dropped := map[uint32]struct{}{}
	for _, d := range droppedIDs {
		dropped[d] = struct{}{}
	}
	wl.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		address := types.BytesToAddress(key[1:])

		model := wl.GetByAddress(address)
		if model != nil && len(model.List) != 0 {
			for _, w := range model.List {
				if _, ok := dropped[w.CandidateId]; ok {
					state.FrozenFunds = append(state.FrozenFunds, types.FrozenFund{
						CandidateID:  0,
						CandidateKey: nil,
						Address:      address,
						Coin:         uint64(w.Coin),
						Value:        w.Value.String(),
					})
					continue
				}
				state.Waitlist = append(state.Waitlist, types.Waitlist{
					CandidateID: uint64(w.CandidateId),
					Owner:       address,
					Coin:        uint64(w.Coin),
					Value:       w.Value.String(),
				})
			}
		}

		return false
	})

	sort.SliceStable(state.Waitlist, func(i, j int) bool {
		return bytes.Compare(state.Waitlist[i].Owner.Bytes(), state.Waitlist[j].Owner.Bytes()) == 1
	})
}

func (wl *WaitList) Commit(db *iavl.MutableTree) error {
	dirty := wl.getOrderedDirty()
	for _, address := range dirty {
		w := wl.getFromMap(address)
		path := append([]byte{mainPrefix}, address.Bytes()...)

		wl.lock.Lock()
		delete(wl.dirty, address)
		wl.lock.Unlock()

		w.lock.RLock()
		if len(w.List) != 0 {
			data, err := rlp.EncodeToBytes(w)
			if err != nil {
				return fmt.Errorf("can't encode object at %s: %v", address.String(), err)
			}
			db.Set(path, data)
		} else {
			db.Remove(path)
			wl.lock.Lock()
			delete(wl.list, address)
			wl.lock.Unlock()
		}
		w.lock.RUnlock()
	}

	return nil
}

func (wl *WaitList) GetByAddress(address types.Address) *Model {
	return wl.get(address)
}

func (wl *WaitList) Get(address types.Address, pubkey types.Pubkey, coin types.CoinID) *Item {
	waitlist := wl.get(address)
	if waitlist == nil {
		return nil
	}

	candidate := wl.bus.Candidates().GetCandidate(pubkey)
	if candidate == nil {
		return nil
	}

	for _, item := range waitlist.List {
		if item.CandidateId == candidate.ID && item.Coin == coin {
			return &item
		}
	}

	return nil
}

func (wl *WaitList) GetByAddressAndPubKey(address types.Address, pubkey types.Pubkey) []Item {
	waitlist := wl.get(address)
	if waitlist == nil {
		return nil
	}

	candidate := wl.bus.Candidates().GetCandidate(pubkey)
	if candidate == nil {
		return nil
	}

	var items []Item
	for _, item := range waitlist.List {
		if item.CandidateId == candidate.ID {
			items = append(items, item)
		}
	}

	return items
}

func (wl *WaitList) AddWaitList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int) {
	w := wl.getOrNew(address)

	candidate := wl.bus.Candidates().GetCandidate(pubkey)
	if candidate == nil {
		log.Panicf("Candidate not found: %s", pubkey.String())
	}

	w.AddToList(candidate.ID, coin, value)
	wl.setToMap(address, w)
	w.markDirty(address)
	wl.bus.Checker().AddCoin(coin, value)
}

func (wl *WaitList) Delete(address types.Address, pubkey types.Pubkey, coin types.CoinID) {
	w := wl.get(address)
	if w == nil || len(w.List) == 0 {
		log.Panicf("Waitlist not found for %s", address.String())
	}

	candidate := wl.bus.Candidates().GetCandidate(pubkey)
	if candidate == nil {
		log.Panicf("Candidate not found: %s", pubkey.String())
	}

	value := big.NewInt(0)

	w.lock.RLock()
	items := make([]Item, 0, len(w.List)-1)
	for _, item := range w.List {
		if item.CandidateId != candidate.ID || item.Coin != coin {
			items = append(items, item)
		} else {
			value.Add(value, item.Value)
		}
	}
	w.List = items
	w.lock.RUnlock()

	wl.markDirty(address)
	wl.setToMap(address, w)
	wl.bus.Checker().AddCoin(coin, big.NewInt(0).Neg(value))
}

func (wl *WaitList) getOrNew(address types.Address) *Model {
	w := wl.get(address)
	if w == nil {
		w = &Model{List: make([]Item, 0), address: address, markDirty: wl.markDirty}
		wl.setToMap(address, w)
	}

	return w
}

func (wl *WaitList) get(address types.Address) *Model {
	if ff := wl.getFromMap(address); ff != nil {
		return ff
	}

	path := append([]byte{mainPrefix}, address.Bytes()...)
	_, enc := wl.immutableTree().Get(path)
	if len(enc) == 0 {
		return nil
	}

	m := new(Model)
	if err := rlp.DecodeBytes(enc, m); err != nil {
		panic(fmt.Sprintf("failed to decode waitlists for address %s: %s", address.String(), err))
	}

	m.address = address
	m.markDirty = wl.markDirty
	wl.setToMap(address, m)

	return m
}

func (wl *WaitList) getFromMap(address types.Address) *Model {
	wl.lock.RLock()
	defer wl.lock.RUnlock()

	return wl.list[address]
}

func (wl *WaitList) setToMap(address types.Address, model *Model) {
	wl.lock.Lock()
	defer wl.lock.Unlock()

	wl.list[address] = model
}

func (wl *WaitList) markDirty(address types.Address) {
	wl.lock.Lock()
	defer wl.lock.Unlock()
	wl.dirty[address] = struct{}{}
}

func (wl *WaitList) getOrderedDirty() []types.Address {
	wl.lock.Lock()
	keys := make([]types.Address, 0, len(wl.dirty))
	for k := range wl.dirty {
		keys = append(keys, k)
	}
	wl.lock.Unlock()

	sort.SliceStable(keys, func(i, j int) bool {
		compare := bytes.Compare(keys[i].Bytes(), keys[j].Bytes())
		return compare == 1
	})

	return keys
}
