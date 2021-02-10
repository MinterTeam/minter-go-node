package coins

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"sync"
)

var minCoinReserve = helpers.BipToPip(big.NewInt(10000))

type Model struct {
	CName      string
	CCrr       uint32
	CMaxSupply *big.Int
	CVersion   types.CoinVersion
	CSymbol    types.CoinSymbol

	Mintable bool
	Burnable bool

	id         types.CoinID
	info       *Info
	symbolInfo *SymbolInfo

	markDirty func(symbol types.CoinID)
	lock      sync.RWMutex

	isDirty   bool
	isCreated bool
}

func (m *Model) Name() string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.CName
}

func (m *Model) Symbol() types.CoinSymbol {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.CSymbol
}

func (m *Model) ID() types.CoinID {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.id
}

func (m *Model) Crr() uint32 {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.CCrr
}

func (m *Model) Volume() *big.Int {
	return big.NewInt(0).Set(m.info.Volume)
}

func (m *Model) Reserve() *big.Int {
	if m.IsToken() {
		return big.NewInt(0)
	}

	m.info.RLock()
	defer m.info.RUnlock()

	return big.NewInt(0).Set(m.info.Reserve)
}

func (m *Model) BaseOrHasReserve() bool {
	return m.ID().IsBaseCoin() || (m.Crr() > 0)
}

func (m *Model) IsToken() bool {
	return !m.BaseOrHasReserve()
}

func (m *Model) Version() uint16 {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.CVersion
}

func (m *Model) IsMintable() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.Mintable
}

func (m *Model) IsBurnable() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.Burnable
}

func (m *Model) SubVolume(amount *big.Int) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.info.Lock()
	m.info.Volume.Sub(m.info.Volume, amount)
	m.info.isDirty = true
	m.info.Unlock()

	m.markDirty(m.id)
}

func (m *Model) AddVolume(amount *big.Int) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.info.Lock()
	m.info.Volume.Add(m.info.Volume, amount)
	m.info.isDirty = true
	m.info.Unlock()

	m.markDirty(m.id)
}

func (m *Model) SubReserve(amount *big.Int) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.info.Lock()
	m.info.Reserve.Sub(m.info.Reserve, amount)
	m.info.isDirty = true
	m.info.Unlock()

	m.markDirty(m.id)
}

func (m *Model) AddReserve(amount *big.Int) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.info.Lock()
	m.info.Reserve.Add(m.info.Reserve, amount)
	m.info.isDirty = true
	m.info.Unlock()

	m.markDirty(m.id)
}

func (m *Model) Mint(amount *big.Int) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.info.Lock()
	m.CMaxSupply.Add(m.CMaxSupply, amount)
	m.isDirty = true
	m.info.Unlock()

	m.markDirty(m.id)
}

func (m *Model) Burn(amount *big.Int) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.info.Lock()
	m.CMaxSupply.Sub(m.CMaxSupply, amount)
	m.isDirty = true
	m.info.Unlock()

	m.markDirty(m.id)
}

func (m *Model) SetVolume(volume *big.Int) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.info.Lock()
	m.info.Volume.Set(volume)
	m.info.isDirty = true
	m.info.Unlock()

	m.markDirty(m.id)
}

func (m *Model) SetReserve(reserve *big.Int) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.info.Lock()
	m.info.Reserve.Set(reserve)
	m.info.isDirty = true
	m.info.Unlock()

	m.markDirty(m.id)
}

func (m *Model) CheckReserveUnderflow(delta *big.Int) error {
	total := big.NewInt(0).Sub(m.Reserve(), delta)

	if total.Cmp(minCoinReserve) == -1 {
		min := big.NewInt(0).Add(minCoinReserve, delta)
		return fmt.Errorf("coin %s reserve is too small (%s, required at least %s)", m.CSymbol.String(), m.Reserve().String(), min.String())
	}

	return nil
}

func (m *Model) IsInfoDirty() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.info.RLock()
	defer m.info.RUnlock()

	return m.info != nil && m.info.isDirty
}

func (m *Model) IsSymbolInfoDirty() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	has := m.symbolInfo != nil

	if !has {
		return false
	}

	m.symbolInfo.RLock()
	defer m.symbolInfo.RUnlock()

	return m.symbolInfo.isDirty
}

func (m *Model) IsDirty() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.isDirty
}

func (m *Model) IsCreated() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.isCreated
}

func (m *Model) MaxSupply() *big.Int {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.CMaxSupply
}

func (m *Model) GetFullSymbol() string {
	if m.Version() == 0 {
		return m.Symbol().String()
	}

	return fmt.Sprintf("%s-%d", m.Symbol(), m.Version())
}

type Info struct {
	Volume  *big.Int
	Reserve *big.Int

	isDirty bool
	sync.RWMutex
}

type SymbolInfo struct {
	COwnerAddress *types.Address

	isDirty bool

	sync.RWMutex
}

func (i *SymbolInfo) setOwnerAddress(address types.Address) {
	i.Lock()
	defer i.Unlock()

	i.COwnerAddress = &address
	i.isDirty = true
}

func (i *SymbolInfo) OwnerAddress() *types.Address {
	i.RLock()
	defer i.RUnlock()

	return i.COwnerAddress
}
