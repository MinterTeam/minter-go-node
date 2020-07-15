package coins

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
)

var minCoinReserve = helpers.BipToPip(big.NewInt(10000))

type Model struct {
	CName      string
	CCrr       uint
	CMaxSupply *big.Int
	CVersion   types.CoinVersion
	CSymbol    types.CoinSymbol

	id         types.CoinID
	info       *Info
	symbolInfo *SymbolInfo

	markDirty func(symbol types.CoinID)

	isDirty   bool
	isCreated bool
}

func (m Model) Name() string {
	return m.CName
}

func (m Model) Symbol() types.CoinSymbol {
	return m.CSymbol
}

func (m Model) ID() types.CoinID {
	return m.id
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

func (m Model) Version() uint16 {
	return m.CVersion
}

func (m *Model) SubVolume(amount *big.Int) {
	m.info.Volume.Sub(m.info.Volume, amount)
	m.markDirty(m.id)
	m.info.isDirty = true
}

func (m *Model) AddVolume(amount *big.Int) {
	m.info.Volume.Add(m.info.Volume, amount)
	m.markDirty(m.id)
	m.info.isDirty = true
}

func (m *Model) SubReserve(amount *big.Int) {
	m.info.Reserve.Sub(m.info.Reserve, amount)
	m.markDirty(m.id)
	m.info.isDirty = true
}

func (m *Model) AddReserve(amount *big.Int) {
	m.info.Reserve.Add(m.info.Reserve, amount)
	m.markDirty(m.id)
	m.info.isDirty = true
}

func (m *Model) SetReserve(reserve *big.Int) {
	m.info.Reserve.Set(reserve)
	m.markDirty(m.id)
	m.info.isDirty = true
}

func (m *Model) SetVolume(volume *big.Int) {
	m.info.Volume.Set(volume)
	m.markDirty(m.id)
	m.info.isDirty = true
}

func (m *Model) CheckReserveUnderflow(delta *big.Int) error {
	total := big.NewInt(0).Sub(m.Reserve(), delta)

	if total.Cmp(minCoinReserve) == -1 {
		min := big.NewInt(0).Add(minCoinReserve, delta)
		return fmt.Errorf("coin %s reserve is too small (%s, required at least %s)", m.CSymbol.String(), m.Reserve().String(), min.String())
	}

	return nil
}

func (m Model) IsInfoDirty() bool {
	return m.info.isDirty
}

func (m Model) IsSymbolInfoDirty() bool {
	return m.symbolInfo != nil && m.symbolInfo.isDirty
}

func (m Model) IsDirty() bool {
	return m.isDirty
}

func (m Model) IsCreated() bool {
	return m.isCreated
}

func (m Model) MaxSupply() *big.Int {
	return m.CMaxSupply
}

func (m Model) GetFullSymbol() string {
	if m.Version() == 0 {
		return m.Symbol().String()
	}

	return fmt.Sprintf("%s-%d", m.Symbol(), m.Version())
}

type Info struct {
	Volume  *big.Int
	Reserve *big.Int

	isDirty bool
}

type SymbolInfo struct {
	COwnerAddress *types.Address

	isDirty bool
}

func (i *SymbolInfo) SetOwnerAddress(address *types.Address) {
	i.COwnerAddress = address
	i.isDirty = true
}

func (i SymbolInfo) OwnerAddress() *types.Address {
	return i.COwnerAddress
}
