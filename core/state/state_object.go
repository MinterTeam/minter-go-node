package state

import (
	"github.com/MinterTeam/minter-go-node/crypto"
	"io"
	"math/big"

	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"sort"
)

// stateAccount represents an Minter account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateAccount struct {
	address types.Address
	data    Account

	deleted bool
	onDirty func(addr types.Address) // Callback method to mark a state object newly dirty
}

// empty returns whether the account is considered empty.
func (s *stateAccount) empty() bool {
	return false
	//return s.data.Nonce == 0 && s.data.Balance.Sign() == 0
}

type Balances struct {
	Data map[types.CoinSymbol]*big.Int
}

type Balance struct {
	Coin   types.CoinSymbol
	Amount *big.Int
}

func (b Balances) getCoins() []types.CoinSymbol {
	keys := make([]types.CoinSymbol, 0, len(b.Data))
	for k, v := range b.Data {
		if v.Cmp(types.Big0) > 0 {
			keys = append(keys, k)
		}
	}

	sort.SliceStable(keys, func(a, b int) bool {
		return big.NewInt(0).SetBytes(keys[a].Bytes()).Cmp(big.NewInt(0).SetBytes(keys[b].Bytes())) == 1
	})

	return keys
}

func (b Balances) EncodeRLP(w io.Writer) error {
	coins := b.getCoins()

	balances := make([]Balance, len(coins))
	for i, k := range coins {
		balances[i] = Balance{k, b.Data[k]}
	}

	return rlp.Encode(w, balances)
}

func (b *Balances) DecodeRLP(s *rlp.Stream) error {

	if b.Data == nil {
		b.Data = make(map[types.CoinSymbol]*big.Int)
	}

	var balances []Balance

	if err := s.Decode(&balances); err != nil {
		return err
	}

	for _, v := range balances {
		b.Data[v.Coin] = v.Amount
	}

	return nil
}

// Account is the Minter consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce        uint64
	Balance      Balances
	MultisigData Multisig
}

type Multisig struct {
	Weights   []uint
	Threshold uint
	Addresses []types.Address
}

func (m *Multisig) Address() types.Address {
	bytes, err := rlp.EncodeToBytes(m)

	if err != nil {
		panic(err)
	}

	var addr types.Address
	copy(addr[:], crypto.Keccak256(bytes)[12:])

	return addr
}

func (m *Multisig) GetWeight(address types.Address) uint {
	for i, addr := range m.Addresses {
		if address == addr {
			return m.Weights[i]
		}
	}

	return 0
}

// newObject creates a state object.
func newObject(address types.Address, data Account, onDirty func(addr types.Address)) *stateAccount {
	if data.Balance.Data == nil {
		data.Balance.Data = make(map[types.CoinSymbol]*big.Int)
	}

	return &stateAccount{
		address: address,
		data:    data,
		onDirty: onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (s *stateAccount) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, s.data)
}

func (s *stateAccount) touch() {
	if s.onDirty != nil {
		s.onDirty(s.Address())
		s.onDirty = nil
	}
}

// AddBalance removes amount from c's balance.
// It is used to add funds to the destination account of a transfer.
func (s *stateAccount) AddBalance(coinSymbol types.CoinSymbol, amount *big.Int) {
	if amount.Sign() == 0 {
		if s.empty() {
			s.touch()
		}

		return
	}
	s.SetBalance(coinSymbol, new(big.Int).Add(s.Balance(coinSymbol), amount))
}

// SubBalance removes amount from c's balance.
// It is used to remove funds from the origin account of a transfer.
func (s *stateAccount) SubBalance(coinSymbol types.CoinSymbol, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(coinSymbol, new(big.Int).Sub(s.Balance(coinSymbol), amount))
}

func (s *stateAccount) SetBalance(coinSymbol types.CoinSymbol, amount *big.Int) {
	s.setBalance(coinSymbol, amount)
}

func (s *stateAccount) setBalance(coinSymbol types.CoinSymbol, amount *big.Int) {
	if s.data.Balance.Data == nil {
		s.data.Balance.Data = make(map[types.CoinSymbol]*big.Int)
	}

	s.data.Balance.Data[coinSymbol] = amount
	if s.onDirty != nil {
		s.onDirty(s.Address())
		s.onDirty = nil
	}
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (s *stateAccount) Address() types.Address {
	return s.address
}

func (s *stateAccount) SetNonce(nonce uint64) {
	s.setNonce(nonce)
}

func (s *stateAccount) setNonce(nonce uint64) {
	s.data.Nonce = nonce
	if s.onDirty != nil {
		s.onDirty(s.Address())
		s.onDirty = nil
	}
}

func (s *stateAccount) Balance(coinSymbol types.CoinSymbol) *big.Int {

	if s.data.Balance.Data == nil {
		return big.NewInt(0)
	}

	if s.data.Balance.Data[coinSymbol] == nil {
		return big.NewInt(0)
	}

	return s.data.Balance.Data[coinSymbol]
}

func (s *stateAccount) Balances() Balances {
	return s.data.Balance
}

func (s *stateAccount) Nonce() uint64 {
	return s.data.Nonce
}

func (s *stateAccount) IsMultisig() bool {
	return len(s.data.MultisigData.Weights) > 0
}

func (s *stateAccount) Multisig() Multisig {
	return s.data.MultisigData
}
