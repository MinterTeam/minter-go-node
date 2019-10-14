package accounts

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
)

const mainPrefix = byte('a')
const coinsPrefix = byte('c')
const balancePrefix = byte('b')

type Accounts struct {
	list  map[types.Address]*Model
	dirty map[types.Address]struct{}

	iavl tree.Tree
}

func NewAccounts(iavl tree.Tree) (*Accounts, error) {
	return &Accounts{iavl: iavl}, nil
}

func (v *Accounts) Commit() error {
	accounts := v.getOrderedDirtyAccounts()
	for _, address := range accounts {
		account := v.list[address]

		// save info (nonce and multisig data)
		if account.isDirty {
			data, err := rlp.EncodeToBytes(account)
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", address[:], err)
			}

			path := []byte{mainPrefix}
			path = append(path, address[:]...)
			v.iavl.Set(path, data)
			account.isDirty = false
		}

		// save coins list
		if account.hasDirtyCoins {
			coins := account.getOrderedCoins()
			coinsList, err := rlp.EncodeToBytes(coins)
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", address[:], err)
			}

			path := []byte{mainPrefix}
			path = append(path, address[:]...)
			path = append(path, coinsPrefix)
			v.iavl.Set(path, coinsList)
			account.hasDirtyCoins = false
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
				path = append(path, coin[:]...)

				balance := account.getBalance(coin)
				if balance.Cmp(big.NewInt(0)) == 0 {
					v.iavl.Remove(path)
				} else {
					v.iavl.Set(path, balance.Bytes())
				}
			}

			account.dirtyBalances = nil
		}
	}

	return nil
}

func (v *Accounts) getOrderedDirtyAccounts() []types.Address {
	keys := make([]types.Address, 0, len(v.dirty))
	for k := range v.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (v *Accounts) AddBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	balance := v.GetBalance(address, coin)
	v.SetBalance(address, coin, big.NewInt(0).Add(balance, amount))
}

func (v *Accounts) GetBalance(address types.Address, coin types.CoinSymbol) *big.Int {
	account := v.getOrNew(address)
	if !account.hasCoin(coin) {
		return big.NewInt(0)
	}

	if _, ok := account.balances[coin]; !ok {
		balance := big.NewInt(0)

		path := []byte{mainPrefix}
		path = append(path, address[:]...)
		path = append(path, balancePrefix)
		path = append(path, coin[:]...)

		_, enc := v.iavl.Get(path)
		if len(enc) != 0 {
			balance = big.NewInt(0).SetBytes(enc)
		}

		account.balances[coin] = balance
	}

	return account.balances[coin]
}

func (v *Accounts) SubBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	balance := v.GetBalance(address, coin)
	v.SetBalance(address, coin, big.NewInt(0).Sub(balance, amount))
}

func (v *Accounts) SetBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	account := v.getOrNew(address)
	account.setBalance(coin, amount)
}

func (v *Accounts) SetNonce(address types.Address, nonce uint64) {
	account := v.getOrNew(address)
	account.setNonce(nonce)
}

func (v *Accounts) Exists(msigAddress types.Address) bool {
	panic("implement me")
}

func (v *Accounts) CreateMultisig(weights []uint, addresses []types.Address, threshold uint) types.Address {
	panic("implement me")
}

func (v *Accounts) get(address types.Address) *Model {
	if account := v.list[address]; account != nil {
		return account
	}

	path := []byte{mainPrefix}
	path = append(path, address[:]...)
	_, enc := v.iavl.Get(path)
	if len(enc) == 0 {
		return nil
	}

	account := &Model{}
	if err := rlp.DecodeBytes(enc, account); err != nil {
		panic(fmt.Sprintf("failed to decode account at address %s: %s", address.String(), err))
		return nil
	}

	account.address = address
	account.balances = map[types.CoinSymbol]*big.Int{}
	account.markDirty = v.markDirty

	// load coins
	path = []byte{mainPrefix}
	path = append(path, address[:]...)
	path = append(path, coinsPrefix)
	_, enc = v.iavl.Get(path)
	if len(enc) != 0 {
		var coins []types.CoinSymbol
		if err := rlp.DecodeBytes(enc, &coins); err != nil {
			panic(fmt.Sprintf("failed to decode coins list at address %s: %s", address.String(), err))
		}

		account.coins = coins
	}

	v.list[address] = account
	return account
}

func (v *Accounts) getOrNew(address types.Address) *Model {
	account := v.get(address)
	if account == nil {
		account = &Model{
			Nonce:     0,
			address:   address,
			coins:     []types.CoinSymbol{},
			balances:  map[types.CoinSymbol]*big.Int{},
			markDirty: v.markDirty,
		}
	}

	return account
}

func (v *Accounts) GetNonce(address types.Address) uint64 {
	account := v.getOrNew(address)

	return account.Nonce
}

func (v *Accounts) GetBalances(address types.Address) map[types.CoinSymbol]*big.Int {
	account := v.getOrNew(address)

	balances := map[types.CoinSymbol]*big.Int{}
	for _, coin := range account.coins {
		balances[coin] = v.GetBalance(address, coin)
	}

	return balances
}

func (v *Accounts) markDirty(addr types.Address) {
	v.dirty[addr] = struct{}{}
}
