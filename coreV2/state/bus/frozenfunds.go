package bus

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type FrozenFund struct {
	Address      types.Address
	CandidateKey *types.Pubkey
	CandidateID  uint32
	Coin         types.CoinID
	Value        *big.Int
}

type FrozenFunds interface {
	AddFrozenFund(uint64, types.Address, types.Pubkey, uint32, types.CoinID, *big.Int)
}
