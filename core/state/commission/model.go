package commission

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
)

type Price struct {
	Coin                    types.CoinID
	PayloadByte             *big.Int
	Send                    *big.Int
	Convert                 *big.Int
	CreateTicker3           *big.Int
	CreateTicker4           *big.Int
	CreateTicker5           *big.Int
	CreateTicker6           *big.Int
	CreateTicker7to10       *big.Int
	RecreateTicker          *big.Int
	DeclareCandidacy        *big.Int
	Delegate                *big.Int
	Unbond                  *big.Int
	RedeemCheck             *big.Int
	ToggleCandidateStatus   *big.Int
	CreateMultisig          *big.Int
	MultisendDelta          *big.Int
	EditCandidate           *big.Int
	SetHaltBlock            *big.Int
	EditTickerOwner         *big.Int
	EditMultisig            *big.Int
	PriceVote               *big.Int
	EditCandidatePublicKey  *big.Int
	AddLiquidity            *big.Int
	RemoveLiquidity         *big.Int
	EditCandidateCommission *big.Int
	MoveStake               *big.Int
	EditTokenEmission       *big.Int
	PriceCommission         *big.Int
	UpdateNetwork           *big.Int
	More                    []*big.Int `rlp:"tail"`
}

func (d *Price) Encode() []byte {
	bytes, err := rlp.EncodeToBytes(d)
	if err != nil {
		panic(err)
	}
	return bytes
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
}

func (m *Model) addVoite(pubkey types.Pubkey) {
	m.Votes = append(m.Votes, pubkey)
	m.markDirty()
}
