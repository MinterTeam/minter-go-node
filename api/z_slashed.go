package api
import (
	"math/big"
)


func TotalSlashed(height int) (*big.Int, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

	totalslashed := cState.App.GetTotalSlashed()
	return totalslashed, nil
}
