package rewards

import "math/big"

func GetRewardForBlock(blockNumber uint64) *big.Int {
	totalBlocksCount := uint64(44512766)
	c := uint64(111)

	if blockNumber > totalBlocksCount + 18 {
		return big.NewInt(0)
	}

	if blockNumber > totalBlocksCount {
		return big.NewInt(1)
	}

	c = (111 * (totalBlocksCount - blockNumber)) / totalBlocksCount + 1

	if blockNumber <= totalBlocksCount * 50 / 100 {
		c = c * 15 / 10
	}

	if c > 111 {
		c = 111
	}

	reward := big.NewInt(int64(c))
	reward.Mul(reward, big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))

	return reward
}
