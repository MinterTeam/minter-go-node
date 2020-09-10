package accounts

import (
	"bytes"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"sort"
)

type Model struct {
	Nonce        uint64
	MultisigData Multisig

	address  types.Address
	coins    []types.CoinID
	balances map[types.CoinID]*big.Int

	hasDirtyCoins bool
	dirtyBalances map[types.CoinID]struct{}
	isDirty       bool // nonce or multisig data

	isNew bool

	markDirty func(types.Address)
}

type Multisig struct {
	Threshold uint
	Weights   []uint
	Addresses []types.Address
}

func CreateMultisigAddress(owner types.Address, nonce uint64) types.Address {
	b, err := rlp.EncodeToBytes(&struct {
		Owner types.Address
		Nonce uint64
	}{Owner: owner, Nonce: nonce})
	if err != nil {
		panic(err)
	}

	var addr types.Address
	copy(addr[:], crypto.Keccak256(b)[12:])

	return addr
}

func (m *Multisig) GetWeight(address types.Address) uint {
	for i, addr := range m.Addresses {
		if addr == address {
			return m.Weights[i]
		}
	}

	return 0
}

func (model *Model) setNonce(nonce uint64) {
	model.Nonce = nonce
	model.isDirty = true
	model.markDirty(model.address)
}

func (model *Model) getBalance(coin types.CoinID) *big.Int {
	return model.balances[coin]
}

func (model *Model) hasDirtyBalances() bool {
	return len(model.dirtyBalances) > 0
}

func (model *Model) isBalanceDirty(coin types.CoinID) bool {
	_, exists := model.dirtyBalances[coin]
	return exists
}

func (model *Model) getOrderedCoins() []types.CoinID {
	keys := make([]types.CoinID, 0, len(model.balances))
	for k := range model.balances {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (model *Model) setBalance(coin types.CoinID, amount *big.Int) {
	if amount.Cmp(big.NewInt(0)) == 0 {
		if !model.hasCoin(coin) {
			return
		}

		var newCoins []types.CoinID
		for _, c := range model.coins {
			if coin == c {
				continue
			}

			newCoins = append(newCoins, c)
		}

		model.hasDirtyCoins = true
		model.coins = newCoins
		model.balances[coin] = amount
		model.dirtyBalances[coin] = struct{}{}
		model.markDirty(model.address)

		return
	}

	if !model.hasCoin(coin) {
		model.hasDirtyCoins = true
		model.coins = append(model.coins, coin)
	}
	model.dirtyBalances[coin] = struct{}{}
	model.markDirty(model.address)
	model.balances[coin] = amount
}

func (model *Model) hasCoin(coin types.CoinID) bool {
	for _, c := range model.coins {
		if c == coin {
			return true
		}
	}

	return false
}

func (model *Model) IsMultisig() bool {
	return len(model.MultisigData.Weights) > 0
}

func (model *Model) Multisig() Multisig {
	return model.MultisigData
}
