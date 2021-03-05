package rewards

import (
	"math/big"

	"github.com/MinterTeam/minter-go-node/helpers"
)

const lastBlock = 43702611
const firstReward = 333
const lastReward = 68

type Reward struct {
	startHeight   uint64
	beforeGenesis *big.Int
}

func NewReward(startHeight uint64) *Reward {
	sHeight := startHeight + 9150000
	r := &Reward{startHeight: sHeight, beforeGenesis: big.NewInt(0)}
	for i := uint64(1); i <= sHeight; i++ {
		r.beforeGenesis.Add(r.beforeGenesis, r.GetRewardForBlock(i))
	}
	return r
}

// GetRewardForBlock returns reward for creation of given block. If there is no reward - returns 0.
func (r *Reward) GetRewardForBlock(blockHeight uint64) *big.Int {
	blockHeight += r.startHeight

	if blockHeight > lastBlock {
		return big.NewInt(0)
	}

	if blockHeight == lastBlock {
		return helpers.BipToPip(big.NewInt(lastReward))
	}

	reward := big.NewInt(firstReward)
	reward.Sub(reward, big.NewInt(int64(blockHeight/200000)))

	if reward.Sign() < 1 {
		return helpers.BipToPip(big.NewInt(1))
	}

	return helpers.BipToPip(reward)
}
