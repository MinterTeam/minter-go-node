package accounts

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	db "github.com/tendermint/tm-db"
	"math/big"
)

type Accounts struct {
	db db.DB
}

func NewAccounts(db db.DB) (*Accounts, error) {
	return &Accounts{db: db}, nil
}

func (v *Accounts) Commit() error {
	panic("implement me")
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

func (v *Accounts) GetBalances(addresses types.Address) interface{} {
	panic("implement me")
}

type Account struct {
	Nonce        uint64
	MultisigData Multisig
}

func (account *Account) IsMultisig() bool {
	panic("implement me")
}

func (account *Account) Multisig() Multisig {
	return account.MultisigData
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
	panic("implement me")
}
