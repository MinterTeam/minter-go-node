package rewards

import (
	"math/big"

	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
)

const lastBlock = 43702611
const firstReward = 333
const lastReward = 68

var startHeight uint64 = 0
var beforeGenesis = big.NewInt(0)

// GetRewardForBlock returns reward for creation of given block. If there is no reward - returns 0.
func GetRewardForBlock(blockHeight uint64) *big.Int {
	blockHeight += startHeight

	if blockHeight > lastBlock {
		return big.NewInt(0)
	}

	if blockHeight == lastBlock {
		return helpers.BipToPip(big.NewInt(lastReward))
	}

	reward := big.NewInt(firstReward)
	reward.Sub(reward, big.NewInt(int64(blockHeight/200000)))

	if reward.Cmp(types.Big0) < 1 {
		return helpers.BipToPip(big.NewInt(1))
	}

	return helpers.BipToPip(reward)
}

// SetStartHeight sets base height for rewards calculations
func SetStartHeight(sHeight uint64) {
	for i := uint64(1); i <= sHeight; i++ {
		beforeGenesis.Add(beforeGenesis, GetRewardForBlock(i))
	}

	startHeight = sHeight
}
