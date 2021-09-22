package bus

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type WaitListItem struct {
	CandidateId uint32
	Coin        types.CoinID
	Value       *big.Int
}

type WaitList interface {
	AddToWaitList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int)
	Delete(address types.Address, pubkey types.Pubkey, coin types.CoinID)
	GetByAddressAndPubKey(address types.Address, pubkey types.Pubkey) []*WaitListItem
}
