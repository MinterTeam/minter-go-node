package rewards

import (
	"math/big"

	"github.com/MinterTeam/minter-go-node/helpers"
)

const lastBlock = 43702611
const firstReward = 333
const lastReward = 68

const TotalEmission = "10000000000000000000000000000"
const premineValue = 200000000

type Reward struct {
	startHeight      uint64
	before           *big.Int
	totalEmissionBig *big.Int
}

func NewReward() *Reward {
	return &Reward{startHeight: 9150000, before: helpers.StringToBigInt("3042199955000000000000000000"), totalEmissionBig: helpers.StringToBigInt(TotalEmission)}
}
func (r *Reward) GetBeforeBlock(blockHeight uint64) *big.Int {
	before := new(big.Int).Set(r.before)
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
