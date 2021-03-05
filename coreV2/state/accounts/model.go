package accounts

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"sort"
	"sync"
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
	lock      sync.RWMutex
}

type Multisig struct {
	Threshold uint32
	Weights   []uint32
	Addresses []types.Address

	lock sync.RWMutex
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

func (m *Multisig) GetWeight(address types.Address) uint32 {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for i, addr := range m.Addresses {
		if addr == address {
			return m.Weights[i]
		}
	}

	return 0
}

func (model *Model) setNonce(nonce uint64) {
	model.lock.Lock()
	defer model.lock.Unlock()

	model.Nonce = nonce
	model.isDirty = true
	model.markDirty(model.address)
}

func (model *Model) getBalance(coin types.CoinID) *big.Int {
	model.lock.RLock()
	defer model.lock.RUnlock()

	return model.balances[coin]
}

func (model *Model) hasDirtyBalances() bool {
	model.lock.RLock()
	defer model.lock.RUnlock()

	return len(model.dirtyBalances) > 0
}

func (model *Model) isBalanceDirty(coin types.CoinID) bool {
	model.lock.RLock()
	defer model.lock.RUnlock()

	_, exists := model.dirtyBalances[coin]
	return exists
}

func (model *Model) getOrderedCoins() []types.CoinID {
	model.lock.RLock()
	keys := make([]types.CoinID, 0, len(model.balances))
	for k := range model.balances {
		keys = append(keys, k)
	}
	model.lock.RUnlock()

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	return keys
}

func (model *Model) setBalance(coin types.CoinID, amount *big.Int) {
	if amount.Cmp(big.NewInt(0)) == 0 {
		if !model.hasCoin(coin) {
			return
		}

		var newCoins []types.CoinID

		model.lock.RLock()
		for _, c := range model.coins {
			if coin == c {
				continue
			}
			newCoins = append(newCoins, c)
		}
		model.lock.RUnlock()

		model.lock.Lock()
		model.hasDirtyCoins = true
		model.coins = newCoins
		model.balances[coin] = amount
		model.dirtyBalances[coin] = struct{}{}
		model.lock.Unlock()

		model.markDirty(model.address)

		return
	}

	if !model.hasCoin(coin) {
		model.lock.Lock()
		model.hasDirtyCoins = true
		model.coins = append(model.coins, coin)
		model.lock.Unlock()
	}

	model.lock.Lock()
	model.dirtyBalances[coin] = struct{}{}
	model.balances[coin] = amount
	model.lock.Unlock()

	model.markDirty(model.address)
}

func (model *Model) hasCoin(coin types.CoinID) bool {
	model.lock.RLock()
	defer model.lock.RUnlock()

	for _, c := range model.coins {
		if c == coin {
			return true
		}
	}

	return false
}

func (model *Model) IsMultisig() bool {
	model.lock.RLock()
	defer model.lock.RUnlock()

	return len(model.MultisigData.Weights) > 0
}

func (model *Model) Multisig() Multisig {
	model.lock.RLock()
	defer model.lock.RUnlock()

	return model.MultisigData
}
