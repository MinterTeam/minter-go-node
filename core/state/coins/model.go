package coins

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
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
	isDeleted bool
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

func (m Model) IsToDelete() bool {
	// Delete coin if reserve is less than 100 bips
	if m.Reserve().Cmp(helpers.BipToPip(big.NewInt(100))) == -1 {
		return true
	}

	// Delete coin if volume is less than 1 coin
	if m.Volume().Cmp(helpers.BipToPip(big.NewInt(1))) == -1 {
		return true
	}

	// Delete coin if price of 1 coin is less than 0.0001 bip
	price := formula.CalculateSaleReturn(m.Volume(), m.Reserve(), m.Crr(), helpers.BipToPip(big.NewInt(1)))
	minPrice := big.NewInt(100000000000000) // 0.0001 bip
	if price.Cmp(minPrice) == -1 {
		return true
	}

	return false
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
