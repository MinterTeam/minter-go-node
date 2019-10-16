package frozen_funds

import (
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
	"sort"
)

const mainPrefix = byte('f')

type FrozenFunds struct {
	list  map[uint64]*Model
	dirty map[uint64]interface{}

	bus  *bus.Bus
	iavl tree.Tree
}

func NewFrozenFunds(stateBus *bus.Bus, iavl tree.Tree) (*FrozenFunds, error) {
	return &FrozenFunds{bus: stateBus, iavl: iavl}, nil
}

func (f *FrozenFunds) Commit() error {
	dirty := f.getOrderedDirty()
	for _, height := range dirty {
		ff := f.list[height]
		delete(f.dirty, height)

		data, err := rlp.EncodeToBytes(ff)
		if err != nil {
			return fmt.Errorf("can't encode object at %d: %v", height, err)
		}

		path := getPath(height)
		f.iavl.Set(path, data)
	}

	return nil
}

func (f *FrozenFunds) GetFrozenFunds(height uint64) *Model {
	return f.get(height)
}

func (f *FrozenFunds) PunishFrozenFundsWithAddress(fromHeight uint64, toHeight uint64, tmAddress types.TmAddress) {
	for cBlock := fromHeight; cBlock <= toHeight; cBlock++ {
		ff := f.get(cBlock)
		if ff == nil {
			continue
		}

		newList := make([]Item, len(ff.List))
		for i, item := range ff.List {
			var pubkey ed25519.PubKeyEd25519
			copy(pubkey[:], item.CandidateKey[:])

			var address [20]byte
			copy(address[:], pubkey.Address().Bytes())

			if tmAddress == address {
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

				// TODO: add event
				//edb.AddEvent(fromBlock, events.SlashEvent{
				//	Address:         item.Address,
				//	Amount:          slashed.Bytes(),
				//	Coin:            item.Coin,
				//	ValidatorPubKey: *item.CandidateKey,
				//})

				item.Value = newValue
				f.bus.Coins().SanitizeCoin(item.Coin)
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
			height: height,
		}
		f.list[height] = ff
	}

	return ff
}

func (f *FrozenFunds) get(height uint64) *Model {
	if ff := f.list[height]; ff != nil {
		return ff
	}

	_, enc := f.iavl.Get(getPath(height))
	if len(enc) == 0 {
		return nil
	}

	ff := &Model{}
	if err := rlp.DecodeBytes(enc, ff); err != nil {
		panic(fmt.Sprintf("failed to decode frozen funds at height %d: %s", height, err))
		return nil
	}

	ff.height = height
	ff.markDirty = f.markDirty

	f.list[height] = ff

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

func getPath(height uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, height)

	return append([]byte{mainPrefix}, b...)
}
