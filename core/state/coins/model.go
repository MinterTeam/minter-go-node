package coins

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Model struct {
	CName      string
	CCrr       uint
	CMaxSupply *big.Int

	symbol    types.CoinSymbol
	info      *Info
	markDirty func(symbol types.CoinSymbol)
	isDirty   bool
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

func (m *Model) SubVolume(amount *big.Int) {
	m.info.Volume.Sub(m.info.Volume, amount)
	m.markDirty(m.symbol)
	m.info.isDirty = true
}

func (m *Model) AddVolume(amount *big.Int) {
	m.info.Volume.Add(m.info.Volume, amount)
	m.markDirty(m.symbol)
	m.info.isDirty = true
}

func (m *Model) SubReserve(amount *big.Int) {
	m.info.Reserve.Sub(m.info.Reserve, amount)
	m.markDirty(m.symbol)
	m.info.isDirty = true
}

func (m *Model) AddReserve(amount *big.Int) {
	m.info.Reserve.Add(m.info.Reserve, amount)
	m.markDirty(m.symbol)
	m.info.isDirty = true
}

func (m *Model) SetReserve(reserve *big.Int) {
	m.info.Reserve.Set(reserve)
	m.markDirty(m.symbol)
	m.info.isDirty = true
}

func (m *Model) SetVolume(volume *big.Int) {
	m.info.Volume.Set(volume)
	m.markDirty(m.symbol)
	m.info.isDirty = true
}

func (m Model) IsInfoDirty() bool {
	return m.info.isDirty
}

func (m Model) IsDirty() bool {
	return m.isDirty
}

func (m Model) MaxSupply() *big.Int {
	return m.CMaxSupply
}

type Info struct {
	Volume  *big.Int
	Reserve *big.Int

	isDirty bool
}
