package app

import "math/big"

type Model struct {
	TotalSlashed *big.Int
	MaxGas       uint64

	markDirty func()
}

func (model *Model) getMaxGas() uint64 {
	return model.MaxGas
}

func (model *Model) setMaxGas(maxGas uint64) {
	if model.MaxGas != maxGas {
		model.markDirty()
	}
	model.MaxGas = maxGas
}

func (model *Model) getTotalSlashed() *big.Int {
	if model.TotalSlashed == nil {
		return big.NewInt(0)
	}

	return model.TotalSlashed
}

func (model *Model) setTotalSlashed(totalSlashed *big.Int) {
	if model.TotalSlashed.Cmp(totalSlashed) != 0 {
		model.markDirty()
	}
	model.TotalSlashed = totalSlashed
}
