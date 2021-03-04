package frozenfunds

import (
	"encoding/binary"
	"fmt"
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
)

const mainPrefix = byte('f')

type RFrozenFunds interface {
	Export(state *types.AppState, height uint64)
	GetFrozenFunds(height uint64) *Model
}

type FrozenFunds struct {
	list  map[uint64]*Model
	dirty map[uint64]interface{}

	bus *bus.Bus
	db  atomic.Value

	lock sync.RWMutex
}

func NewFrozenFunds(stateBus *bus.Bus, db *iavl.ImmutableTree) *FrozenFunds {
	immutableTree := atomic.Value{}
	if db != nil {
		immutableTree.Store(db)
	}
	frozenFunds := &FrozenFunds{bus: stateBus, db: immutableTree, list: map[uint64]*Model{}, dirty: map[uint64]interface{}{}}
	frozenFunds.bus.SetFrozenFunds(NewBus(frozenFunds))

	return frozenFunds
}

func (f *FrozenFunds) immutableTree() *iavl.ImmutableTree {
	db := f.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (f *FrozenFunds) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	f.db.Store(immutableTree)
}
func (f *FrozenFunds) Commit(db *iavl.MutableTree) error {
	dirty := f.getOrderedDirty()
	for _, height := range dirty {
		ff := f.getFromMap(height)
		path := getPath(height)

		f.lock.Lock()
		delete(f.dirty, height)
		f.lock.Unlock()

		ff.lock.RLock()
		if ff.deleted {

			f.lock.Lock()
			delete(f.list, height)
			f.lock.Unlock()

			db.Remove(path)
		} else {
			data, err := rlp.EncodeToBytes(ff)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", height, err)
			}

			db.Set(path, data)
		}
		ff.lock.RUnlock()
	}

	return nil
}

func (f *FrozenFunds) GetFrozenFunds(height uint64) *Model {
	return f.get(height)
}

func (f *FrozenFunds) PunishFrozenFundsWithID(fromHeight uint64, toHeight uint64, candidateID uint32) {
	for cBlock := fromHeight; cBlock <= toHeight; cBlock++ {
		ff := f.get(cBlock)
		if ff == nil {
			continue
		}

		ff.lock.Lock()
		newList := make([]Item, len(ff.List))
		for i, item := range ff.List {
			if item.CandidateID == candidateID {
				newValue := big.NewInt(0).Set(item.Value)
				newValue.Mul(newValue, big.NewInt(95))
				newValue.Div(newValue, big.NewInt(100))

				slashed := big.NewInt(0).Set(item.Value)
				slashed.Sub(slashed, newValue)

				if !item.Coin.IsBaseCoin() {
					coin := f.bus.Coins().GetCoin(item.Coin)
					ret := formula.CalculateSaleReturn(coin.Volume, coin.Reserve, coin.Crr, slashed)
					f.bus.Coins().SubCoinVolume(item.Coin, slashed)
					f.bus.Coins().SubCoinReserve(item.Coin, ret)
					f.bus.App().AddTotalSlashed(ret)
				} else {
					f.bus.App().AddTotalSlashed(slashed)
				}

				f.bus.Checker().AddCoin(item.Coin, new(big.Int).Neg(slashed))

				f.bus.Events().AddEvent(&eventsdb.SlashEvent{
					Address:         item.Address,
					Amount:          slashed.String(),
					Coin:            uint64(item.Coin),
					ValidatorPubKey: *item.CandidateKey,
				})

				item.Value = newValue
			}

			newList[i] = item
		}
		ff.List = newList
		ff.lock.Unlock()

		f.markDirty(cBlock)
	}
}

func (f *FrozenFunds) GetOrNew(height uint64) *Model {
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

func (f *FrozenFunds) get(height uint64) *Model {
	if ff := f.getFromMap(height); ff != nil {
		return ff
	}

	_, enc := f.immutableTree().Get(getPath(height))
	if len(enc) == 0 {
		return nil
	}

	ff := &Model{}
	if err := rlp.DecodeBytes(enc, ff); err != nil {
		panic(fmt.Sprintf("failed to decode frozen funds at height %d: %s", height, err))
	}

	ff.height = height
	ff.markDirty = f.markDirty

	f.setToMap(height, ff)

	return ff
}

func (f *FrozenFunds) markDirty(height uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.dirty[height] = struct{}{}
}

func (f *FrozenFunds) getOrderedDirty() []uint64 {
	f.lock.Lock()
	keys := make([]uint64, 0, len(f.dirty))
	for k := range f.dirty {
		keys = append(keys, k)
	}
	f.lock.Unlock()

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func (f *FrozenFunds) AddFund(height uint64, address types.Address, pubkey types.Pubkey, candidateId uint32, coin types.CoinID, value *big.Int, moveToCandidate *uint32) {
	f.GetOrNew(height).addFund(address, pubkey, candidateId, coin, value, moveToCandidate)
	f.bus.Checker().AddCoin(coin, value)
}

func (f *FrozenFunds) Delete(height uint64) {
	ff := f.get(height)
	if ff == nil {
		return
	}

	ff.delete()

	for _, fund := range ff.List {
		f.bus.Checker().AddCoin(fund.Coin, big.NewInt(0).Neg(fund.Value))
	}
}

func (f *FrozenFunds) Export(state *types.AppState, height uint64) {
	for i := height; i <= height+types.GetUnbondPeriodWithChain(types.ChainMainnet); i++ {
		frozenFunds := f.get(i)
		if frozenFunds == nil {
			continue
		}

		frozenFunds.lock.RLock()
		for _, frozenFund := range frozenFunds.List {
			state.FrozenFunds = append(state.FrozenFunds, types.FrozenFund{
				Height:       i,
				Address:      frozenFund.Address,
				CandidateKey: frozenFund.CandidateKey,
				CandidateID:  uint64(frozenFund.CandidateID),
				Coin:         uint64(frozenFund.Coin),
				Value:        frozenFund.Value.String(),
			})
		}
		frozenFunds.lock.RUnlock()
	}
}

func (f *FrozenFunds) getFromMap(height uint64) *Model {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.list[height]
}

func (f *FrozenFunds) setToMap(height uint64, model *Model) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.list[height] = model
}

func getPath(height uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, height)

	return append([]byte{mainPrefix}, b...)
}
