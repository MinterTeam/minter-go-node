package accounts

import (
	"bytes"
	"fmt"
	"sync/atomic"

	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"

	"math/big"
	"sort"
	"sync"
)

const mainPrefix = byte('a')
const coinsPrefix = byte('c')
const balancePrefix = byte('b')

type RAccounts interface {
	// Deprecated
	ExportV1(state *types.AppState, value *big.Int) (map[types.CoinID]*big.Int, map[types.CoinID]*coins.MaxCoinVolume)

	Export(state *types.AppState)
	GetAccount(address types.Address) *Model
	GetNonce(address types.Address) uint64
	GetBalance(address types.Address, coin types.CoinID) *big.Int
	GetBalances(address types.Address) []Balance
	ExistsMultisig(msigAddress types.Address) bool
}

type Accounts struct {
	list  map[types.Address]*Model
	dirty map[types.Address]struct{}

	db  atomic.Value
	bus *bus.Bus

	lock sync.RWMutex
}

type Balance struct {
	Coin  bus.Coin
	Value *big.Int
}

func NewAccounts(stateBus *bus.Bus, db *iavl.ImmutableTree) *Accounts {
	immutableTree := atomic.Value{}
	if db != nil {
		immutableTree.Store(db)
	}
	accounts := &Accounts{db: immutableTree, bus: stateBus, list: map[types.Address]*Model{}, dirty: map[types.Address]struct{}{}}
	accounts.bus.SetAccounts(NewBus(accounts))

	return accounts
}

func (a *Accounts) immutableTree() *iavl.ImmutableTree {
	db := a.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (a *Accounts) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	a.db.Store(immutableTree)
}

func (a *Accounts) Commit(db *iavl.MutableTree, version int64) error {
	accounts := a.getOrderedDirtyAccounts()
	for _, address := range accounts {
		account := a.getFromMap(address)
		a.lock.Lock()
		delete(a.dirty, address)
		a.lock.Unlock()

		// save info (nonce and multisig data)
		if a.IsNewOrDirty(account) {
			account.lock.Lock()
			account.isDirty = false
			account.isNew = false
			data, err := rlp.EncodeToBytes(account)
			account.lock.Unlock()
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", address[:], err)
			}
			if len(data) == 0 {
				continue
			}

			path := []byte{mainPrefix}
			path = append(path, address[:]...)
			db.Set(path, data)
		}

		// save coins list
		if a.HasDirtyCoins(account) {
			account.lock.Lock()
			account.hasDirtyCoins = false
			coinsList, err := rlp.EncodeToBytes(account.coins)
			account.lock.Unlock()
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", address[:], err)
			}

			path := []byte{mainPrefix}
			path = append(path, address[:]...)
			path = append(path, coinsPrefix)
			db.Set(path, coinsList)
		}

		// save balances
		if account.hasDirtyBalances() {
			coins := account.getOrderedCoins()
			for _, coin := range coins {
				if !account.isBalanceDirty(coin) {
					continue
				}

				path := []byte{mainPrefix}
				path = append(path, address[:]...)
				path = append(path, balancePrefix)
				path = append(path, coin.Bytes()...)

				balance := account.getBalance(coin)
				switch balance.Sign() {
				case 0:
					db.Remove(path)
				case 1:
					db.Set(path, balance.Bytes())
				case -1:
					if version < 4415830 && types.CurrentChainID == types.ChainMainnet {
						db.Set(path, balance.Bytes())
					} else {
						panic(fmt.Sprintf("Address %s has negative balance of CoinID %d: %s", account.address.String(), coin.Uint32(), balance))
					}
				}
			}

			account.lock.Lock()
			account.dirtyBalances = map[types.CoinID]struct{}{}
			account.lock.Unlock()
		}
	}

	return nil
}

func (a *Accounts) HasDirtyCoins(account *Model) bool {
	account.lock.RLock()
	defer account.lock.RUnlock()

	return account.hasDirtyCoins
}

func (a *Accounts) IsNewOrDirty(account *Model) bool {
	account.lock.RLock()
	defer account.lock.RUnlock()

	return account.isDirty || account.isNew
}

func (a *Accounts) getOrderedDirtyAccounts() []types.Address {
	a.lock.RLock()
	keys := make([]types.Address, 0, len(a.dirty))
	for k := range a.dirty {
		keys = append(keys, k)
	}
	a.lock.RUnlock()

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (a *Accounts) AddBalance(address types.Address, coin types.CoinID, amount *big.Int) {
	balance := a.GetBalance(address, coin)
	a.SetBalance(address, coin, big.NewInt(0).Add(balance, amount))
}

func (a *Accounts) GetBalance(address types.Address, coin types.CoinID) *big.Int {
	account := a.getOrNew(address)
	if !account.hasCoin(coin) {
		return big.NewInt(0)
	}

	account.lock.RLock()
	balance, ok := account.balances[coin]
	account.lock.RUnlock()
	if !ok {
		balance = big.NewInt(0)

		path := []byte{mainPrefix}
		path = append(path, address[:]...)
		path = append(path, balancePrefix)
		path = append(path, coin.Bytes()...)

		_, enc := a.immutableTree().Get(path)
		if len(enc) != 0 {
			balance = big.NewInt(0).SetBytes(enc)
		}

		account.lock.Lock()
		account.balances[coin] = balance
		account.lock.Unlock()
	}

	return big.NewInt(0).Set(balance)
}

func (a *Accounts) SubBalance(address types.Address, coin types.CoinID, amount *big.Int) {
	balance := big.NewInt(0).Sub(a.GetBalance(address, coin), amount)
	a.SetBalance(address, coin, balance)
}

func (a *Accounts) SetBalance(address types.Address, coin types.CoinID, amount *big.Int) {
	account := a.getOrNew(address)
	oldBalance := a.GetBalance(address, coin)
	a.bus.Checker().AddCoin(coin, big.NewInt(0).Sub(amount, oldBalance))

	account.setBalance(coin, amount)
}

func (a *Accounts) SetNonce(address types.Address, nonce uint64) {
	account := a.getOrNew(address)
	account.setNonce(nonce)
}

func (a *Accounts) ExistsMultisig(msigAddress types.Address) bool {
	acc := a.get(msigAddress)
	if acc == nil {
		return false
	}

	if acc.IsMultisig() {
		return true
	}

	acc.lock.RLock()
	defer acc.lock.RUnlock()

	if acc.Nonce > 0 {
		return true
	}

	return false
}

func (a *Accounts) CreateMultisig(weights []uint32, addresses []types.Address, threshold uint32, address types.Address) types.Address {
	account := a.get(address)
	if account == nil {
		account = &Model{
			Nonce:         0,
			address:       address,
			coins:         []types.CoinID{},
			balances:      map[types.CoinID]*big.Int{},
			markDirty:     a.markDirty,
			isNew:         true,
			dirtyBalances: map[types.CoinID]struct{}{},
		}
	}

	account.MultisigData = Multisig{
		Weights:   weights,
		Threshold: threshold,
		Addresses: addresses,
	}
	account.markDirty(account.address)
	account.isDirty = true
	a.setToMap(address, account)

	return address
}

func (a *Accounts) EditMultisig(threshold uint32, weights []uint32, addresses []types.Address, address types.Address) types.Address {
	account := a.get(address)

	account.lock.Lock()
	account.MultisigData = Multisig{
		Threshold: threshold,
		Weights:   weights,
		Addresses: addresses,
	}
	account.isDirty = true
	account.lock.Unlock()

	account.markDirty(account.address)
	a.setToMap(address, account)

	return address
}

func (a *Accounts) get(address types.Address) *Model {
	if account := a.getFromMap(address); account != nil {
		return account
	}

	path := []byte{mainPrefix}
	path = append(path, address[:]...)
	_, enc := a.immutableTree().Get(path)
	if len(enc) == 0 {
		return nil
	}

	account := &Model{}
	if err := rlp.DecodeBytes(enc, account); err != nil {
		panic(fmt.Sprintf("failed to decode account at address %s: %s", address.String(), err))
	}

	account.address = address
	account.balances = map[types.CoinID]*big.Int{}
	account.markDirty = a.markDirty
	account.dirtyBalances = map[types.CoinID]struct{}{}

	// load coins
	path = []byte{mainPrefix}
	path = append(path, address[:]...)
	path = append(path, coinsPrefix)
	_, enc = a.immutableTree().Get(path)
	if len(enc) != 0 {
		var coins []types.CoinID
		if err := rlp.DecodeBytes(enc, &coins); err != nil {
			panic(fmt.Sprintf("failed to decode coins list at address %s: %s", address.String(), err))
		}

		account.coins = coins
	}

	a.setToMap(address, account)
	return account
}

func (a *Accounts) getOrNew(address types.Address) *Model {
	account := a.get(address)
	if account == nil {
		account = &Model{
			Nonce:         0,
			address:       address,
			coins:         []types.CoinID{},
			balances:      map[types.CoinID]*big.Int{},
			markDirty:     a.markDirty,
			dirtyBalances: map[types.CoinID]struct{}{},
			isNew:         true,
		}
		a.setToMap(address, account)
	}

	return account
}

func (a *Accounts) GetNonce(address types.Address) uint64 {
	account := a.getOrNew(address)

	account.lock.RLock()
	defer account.lock.RUnlock()

	return account.Nonce
}

func (a *Accounts) GetBalances(address types.Address) []Balance {
	account := a.getOrNew(address)

	account.lock.RLock()
	coins := account.coins
	account.lock.RUnlock()

	balances := make([]Balance, len(coins))
	for key, id := range coins {
		balances[key] = Balance{
			Coin:  *a.bus.Coins().GetCoin(id),
			Value: a.GetBalance(address, id),
		}
	}

	return balances
}

func (a *Accounts) markDirty(addr types.Address) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.dirty[addr] = struct{}{}
}

func (a *Accounts) Export(state *types.AppState) {
	a.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		addressPath := key[1:]
		if len(addressPath) > types.AddressLength {
			return false
		}

		address := types.BytesToAddress(addressPath)
		account := a.get(address)

		var balance []types.Balance
		for _, b := range a.GetBalances(account.address) {
			if b.Value.Sign() != 1 {
				continue
			}
			balance = append(balance, types.Balance{
				Coin:  uint64(b.Coin.ID),
				Value: b.Value.String(),
			})
		}

		sort.SliceStable(balance, func(i, j int) bool {
			return balance[i].Coin < balance[j].Coin
		})

		acc := types.Account{
			Address: account.address,
			Balance: balance,
			Nonce:   account.Nonce,
		}

		if account.IsMultisig() {
			var weights []uint64
			for _, weight := range account.MultisigData.Weights {
				weights = append(weights, uint64(weight))
			}
			acc.MultisigData = &types.Multisig{
				Weights:   weights,
				Threshold: uint64(account.MultisigData.Threshold),
				Addresses: account.MultisigData.Addresses,
			}
		}

		if len(acc.Balance) == 0 && acc.Nonce == 0 && acc.MultisigData == nil {
			return false
		}

		state.Accounts = append(state.Accounts, acc)

		return false
	})
}

func (a *Accounts) GetAccount(address types.Address) *Model {
	return a.getOrNew(address)
}

func (a *Accounts) getFromMap(address types.Address) *Model {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.list[address]
}

func (a *Accounts) setToMap(address types.Address, model *Model) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.list[address] = model
}
