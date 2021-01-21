package commission

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
)

type Price struct {
	PayloadByte             *big.Int
	Send                    *big.Int
	Convert                 *big.Int
	CreateTicker3           *big.Int
	CreateTicker4           *big.Int
	CreateTicker5           *big.Int
	CreateTicker6           *big.Int
	CreateTicker7to10       *big.Int
	Recreate                *big.Int
	DeclareCandidacy        *big.Int
	Delegate                *big.Int
	Unbond                  *big.Int
	RedeemCheck             *big.Int
	ToggleCandidateStatus   *big.Int
	CreateMultisig          *big.Int
	MultisendDelta          *big.Int
	EditCandidate           *big.Int
	SetHaltBlock            *big.Int
	EditCoinOwner           *big.Int
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
	Coin                    types.CoinID
}

func (d *Price) Encode() []byte {
	bytes, err := rlp.EncodeToBytes(d)
	if err != nil {
		panic(err)
	}
	return bytes
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
