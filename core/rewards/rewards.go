package rewards

import "math/big"

func GetRewardForBlock(block int64) *big.Int {
	return big.NewInt(1e8)
}
