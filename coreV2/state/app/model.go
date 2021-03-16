package app

import (
	"math/big"
	"sync"
)

type Model struct {
	TotalSlashed *big.Int
	CoinsCount   uint32
	MaxGas       uint64
	Version      string

	markDirty func()
	mx        sync.RWMutex
}

func (model *Model) getMaxGas() uint64 {
	model.mx.RLock()
	defer model.mx.RUnlock()

	return model.MaxGas
}

func (model *Model) setMaxGas(maxGas uint64) {
	model.mx.Lock()
	defer model.mx.Unlock()

	if model.MaxGas != maxGas {
		model.markDirty()
	}
	model.MaxGas = maxGas
}

func (model *Model) getTotalSlashed() *big.Int {
	model.mx.RLock()
	defer model.mx.RUnlock()

	if model.TotalSlashed == nil {
		return big.NewInt(0)
	}

	return model.TotalSlashed
}

func (model *Model) setTotalSlashed(totalSlashed *big.Int) {
	model.mx.Lock()
	defer model.mx.Unlock()

	if model.TotalSlashed.Cmp(totalSlashed) != 0 {
		model.markDirty()
	}
	model.TotalSlashed = totalSlashed
}

func (model *Model) getCoinsCount() uint32 {
	model.mx.RLock()
	defer model.mx.RUnlock()

	return model.CoinsCount
}

func (model *Model) setCoinsCount(count uint32) {
	model.mx.Lock()
	defer model.mx.Unlock()

	if model.CoinsCount != count {
		model.markDirty()
	}

	model.CoinsCount = count
}

func (model *Model) setVersion(version string) {
	model.mx.Lock()
	defer model.mx.Unlock()

	if model.Version != version {
		model.markDirty()
	}

	model.Version = version
}

func (model *Model) getVersion() string {
	model.mx.RLock()
	defer model.mx.RUnlock()

	return model.Version
}
