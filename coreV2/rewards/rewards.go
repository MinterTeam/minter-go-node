package rewards

import (
	"math/big"

	"github.com/MinterTeam/minter-go-node/helpers"
)

const lastBlock = 43702611
const firstReward = 333
const lastReward = 68

const TotalEmission = "9381399999000000000000000000"

type Reward struct {
	startHeight      uint64
	totalEmissionBig *big.Int
}

func NewReward() *Reward {
	return &Reward{startHeight: 9150000, totalEmissionBig: helpers.StringToBigInt(TotalEmission)}
}
func (r *Reward) GetBeforeBlock(blockHeight uint64) *big.Int {
	before := helpers.StringToBigInt("2423599954000000000000000000")
	for i := uint64(1); i <= blockHeight; i++ {
		before.Add(before, r.GetRewardForBlock(i))
	}
	return before
}
func (r *Reward) TotalEmissionBig() *big.Int {
	return r.totalEmissionBig
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
