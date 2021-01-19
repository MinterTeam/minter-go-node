package commission

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
)

type Price struct {
	Send                   *big.Int
	SellCoin               *big.Int
	SellAllCoin            *big.Int
	BuyCoin                *big.Int
	CreateCoin             *big.Int
	DeclareCandidacy       *big.Int
	Delegate               *big.Int
	Unbond                 *big.Int
	RedeemCheck            *big.Int
	SetCandidateOnline     *big.Int
	SetCandidateOffline    *big.Int
	CreateMultisig         *big.Int
	Multisend              *big.Int
	EditCandidate          *big.Int
	SetHaltBlock           *big.Int
	RecreateCoin           *big.Int
	EditCoinOwner          *big.Int
	EditMultisig           *big.Int
	PriceVote              *big.Int
	EditCandidatePublicKey *big.Int
	AddLiquidity           *big.Int
	RemoveLiquidity        *big.Int
	SellSwapPool           *big.Int
	BuySwapPool            *big.Int
	SellAllSwapPool        *big.Int
	EditCommission         *big.Int
	MoveStake              *big.Int
	MintToken              *big.Int
	BurnToken              *big.Int
	CreateToken            *big.Int
	RecreateToken          *big.Int
	PriceCommission        *big.Int
	UpdateNetwork          *big.Int
	Coin                   types.CoinID
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
