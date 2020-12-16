package frozenfunds

import (
	"encoding/binary"
	"fmt"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
	"sync"
)

const mainPrefix = byte('f')

type RFrozenFunds interface {
	Export(state *types.AppState, height uint64)
	GetFrozenFunds(height uint64) *Model
}

type FrozenFunds struct {
	list  map[uint64]*Model
	dirty map[uint64]interface{}

	bus  *bus.Bus
	iavl tree.MTree

	lock sync.RWMutex
}

func NewFrozenFunds(stateBus *bus.Bus, iavl tree.MTree) (*FrozenFunds, error) {
	frozenfunds := &FrozenFunds{bus: stateBus, iavl: iavl, list: map[uint64]*Model{}, dirty: map[uint64]interface{}{}}
	frozenfunds.bus.SetFrozenFunds(NewBus(frozenfunds))

	return frozenfunds, nil
}

func (f *FrozenFunds) Commit() error {
	dirty := f.getOrderedDirty()
	for _, height := range dirty {
		ff := f.getFromMap(height)

		f.lock.Lock()
		delete(f.dirty, height)
		delete(f.list, height)
		f.lock.Unlock()

		path := getPath(height)

		if ff.deleted {
			f.lock.Lock()
			delete(f.list, height)
			f.lock.Unlock()

			f.iavl.Remove(path)
		} else {
			data, err := rlp.EncodeToBytes(ff)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", height, err)
			}

			f.iavl.Set(path, data)
		}
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

				f.bus.Events().AddEvent(uint32(fromHeight), &eventsdb.SlashEvent{
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

	_, enc := f.iavl.Get(getPath(height))
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
	f.dirty[height] = struct{}{}
}

func (f *FrozenFunds) getOrderedDirty() []uint64 {
	keys := make([]uint64, 0, len(f.dirty))
	for k := range f.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func (f *FrozenFunds) AddFund(height uint64, address types.Address, pubkey types.Pubkey, candidateId uint32, coin types.CoinID, value *big.Int) {
	f.GetOrNew(height).addFund(address, pubkey, candidateId, coin, value)
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
	for i := height; i <= height+candidates.UnbondPeriod; i++ {
		frozenFunds := f.get(i)
		if frozenFunds == nil {
			continue
		}

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
