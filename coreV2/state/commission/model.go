package commission

import (
	"math/big"
	"sync"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
)

type Price struct {
	Coin                    types.CoinID
	PayloadByte             *big.Int
	Send                    *big.Int
	BuyBancor               *big.Int
	SellBancor              *big.Int
	SellAllBancor           *big.Int
	BuyPoolBase             *big.Int
	BuyPoolDelta            *big.Int
	SellPoolBase            *big.Int
	SellPoolDelta           *big.Int
	SellAllPoolBase         *big.Int
	SellAllPoolDelta        *big.Int
	CreateTicker3           *big.Int
	CreateTicker4           *big.Int
	CreateTicker5           *big.Int
	CreateTicker6           *big.Int
	CreateTicker7to10       *big.Int
	CreateCoin              *big.Int
	CreateToken             *big.Int
	RecreateCoin            *big.Int
	RecreateToken           *big.Int
	DeclareCandidacy        *big.Int
	Delegate                *big.Int
	Unbond                  *big.Int
	RedeemCheck             *big.Int
	SetCandidateOn          *big.Int
	SetCandidateOff         *big.Int
	CreateMultisig          *big.Int
	MultisendBase           *big.Int
	MultisendDelta          *big.Int
	EditCandidate           *big.Int
	SetHaltBlock            *big.Int
	EditTickerOwner         *big.Int
	EditMultisig            *big.Int
	EditCandidatePublicKey  *big.Int
	CreateSwapPool          *big.Int
	AddLiquidity            *big.Int
	RemoveLiquidity         *big.Int
	EditCandidateCommission *big.Int
	BurnToken               *big.Int
	MintToken               *big.Int
	VoteCommission          *big.Int
	VoteUpdate              *big.Int

	// FailedTx, AddLimitOrder, RemoveLimitOrder *big.Int
	More []*big.Int `rlp:"tail"`
}

func (d *Price) Encode() []byte {
	bytes, err := rlp.EncodeToBytes(d)
	if err != nil {
		panic(err)
	}
	return bytes
}

func (d *Price) FailedTxPrice() *big.Int {
	if len(d.More) > 0 {
		return d.More[0]
	}
	return d.Send
}

func (d *Price) AddLimitOrderPrice() *big.Int {
	if len(d.More) > 1 {
		return d.More[1]
	}
	return d.CreateSwapPool
}

func (d *Price) RemoveLimitOrderPrice() *big.Int {
	if len(d.More) > 2 {
		return d.More[2]
	}
	return d.RemoveLiquidity
}

func Decode(s string) *Price {
	var p Price
	err := rlp.DecodeBytes([]byte(s), &p)
	if err != nil {
		panic(err)
	}
	return &p
}

type Model struct {
	Votes []types.Pubkey
	Price string

	height    uint64
	markDirty func()

	lock sync.Mutex
}

func (m *Model) addVote(pubkey types.Pubkey) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Votes = append(m.Votes, pubkey)
	m.markDirty()
}
