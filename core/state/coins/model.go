package coins

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Model struct {
	CName string
	CCrr  uint

	symbol types.CoinSymbol
	info   *Info
}

func (m Model) Name() string {
	return m.CName
}

func (m Model) Symbol() types.CoinSymbol {
	return m.symbol
}

func (m Model) Crr() uint {
	return m.CCrr
}

func (m Model) Volume() *big.Int {
	return big.NewInt(0).Set(m.info.Volume)
}

func (m Model) Reserve() *big.Int {
	return big.NewInt(0).Set(m.info.Reserve)
}

type Info struct {
	Volume  *big.Int
	Reserve *big.Int
}
