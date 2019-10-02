package accounts

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
)

const mainPrefix = byte('a')
const coinsPrefix = byte('c')
const balancePrefix = byte('b')

type Accounts struct {
	list map[types.Address]*Account

	iavl tree.Tree
}

func NewAccounts(iavl tree.Tree) (*Accounts, error) {
	return &Accounts{iavl: iavl}, nil
}

func (v *Accounts) Commit() error {
	keys := v.getOrderedListKeys()

	for _, address := range keys {
		account := v.list[address]
		coins := account.getOrderedCoins()

		// save coins list
		if account.hasDirtyCoins {
			coinsList, err := rlp.EncodeToBytes(coins)
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", address[:], err)
			}

			path := []byte(mainPrefix)
			path = append(path, address[:]...)
			path = append(path, coinsPrefix)
			v.iavl.Set(path, coinsList)
		}

		// save balances
		if account.hasDirtyBalances() {
			for _, coin := range coins {
				if !account.isBalanceDirty(coin) {
					continue
				}

				path := []byte(mainPrefix)
				path = append(path, address[:]...)
				path = append(path, balancePrefix)
				v.iavl.Set(path, account.GetBalance(coin).Bytes())
			}
		}

		// save info
		if account.isDirty {
			data, err := rlp.EncodeToBytes(account)
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", address[:], err)
			}

			path := []byte(mainPrefix)
			path = append(path, address[:]...)
			v.iavl.Set(path, data)
		}
	}

	return nil
}

func (v *Accounts) getOrderedListKeys() []types.Address {
	keys := make([]types.Address, 0, len(v.list))
	for k := range v.list {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (v *Accounts) AddBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (v *Accounts) GetBalance(address types.Address, coin types.CoinSymbol) *big.Int {
	panic("implement me")
}

func (v *Accounts) SubBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (v *Accounts) SetNonce(address types.Address, nonce uint64) {
	panic("implement me")
}

func (v *Accounts) Exists(msigAddress types.Address) bool {
	panic("implement me")
}

func (v *Accounts) CreateMultisig(weights []uint, addresses []types.Address, threshold uint) types.Address {
	panic("implement me")
}

func (v *Accounts) GetOrNew(addresses types.Address) *Account {
	panic("implement me")
}

func (v *Accounts) GetNonce(addresses types.Address) uint64 {
	panic("implement me")
}

func (v *Accounts) GetBalances(addresses types.Address) map[types.CoinSymbol]*big.Int {
	panic("implement me")
}

type Account struct {
	Nonce        uint64
	MultisigData *Multisig

	address  types.Address
	balances map[types.CoinSymbol]*big.Int

	hasDirtyCoins bool
	dirtyBalances map[types.CoinSymbol]struct{}
	isDirty       bool
}

func (account *Account) GetBalance(coin types.CoinSymbol) *big.Int {
	return account.balances[coin]
}

func (account *Account) hasDirtyBalances() bool {
	return len(account.dirtyBalances) > 0
}

func (account *Account) isBalanceDirty(coin types.CoinSymbol) bool {
	_, exists := account.dirtyBalances[coin]
	return exists
}

func (account *Account) getOrderedCoins() []types.CoinSymbol {
	keys := make([]types.CoinSymbol, 0, len(account.balances))
	for k := range account.balances {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (account *Account) IsMultisig() bool {
	panic("implement me")
}

func (account *Account) Multisig() *Multisig {
	return account.MultisigData
}

type Multisig struct {
	Weights   []uint
	Threshold uint
	Addresses []types.Address
}

func (m *Multisig) Address() types.Address {
	b, err := rlp.EncodeToBytes(m)

	if err != nil {
		panic(err)
	}

	var addr types.Address
	copy(addr[:], crypto.Keccak256(b)[12:])

	return addr
}

func (m *Multisig) GetWeight(address types.Address) uint {
	panic("implement me")
}
