package api
import (
	"math/big"
)


func AllSlashed(height int) (**big.Int, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

	slashEd := cState.App.GetTotalSlashed()
	return &slashEd, nil
}
