package rewards

import (
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"testing"
)

type Results struct {
	Block  uint64
	Result *big.Int
}

func TestGetRewardForBlock(t *testing.T) {
	data := []Results{
		{
			Block:  1,
			Result: helpers.BipToPip(big.NewInt(333)),
		},
		{
			Block:  43702611,
			Result: helpers.BipToPip(big.NewInt(68)),
		},
		{
			Block:  36600000,
			Result: helpers.BipToPip(big.NewInt(150)),
		},
	}

	for _, item := range data {
		result := GetRewardForBlock(item.Block)

		if result.Cmp(item.Result) != 0 {
			t.Errorf("GetRewardForBlock result is not correct. Expected %d, got %d", item.Result, result)
		}
	}
}

func TestTotalRewardsCount(t *testing.T) {
	total := big.NewInt(0)
	target := helpers.BipToPip(big.NewInt(9800000000))

	for i := uint64(1); i <= 43703000; i++ {
		total.Add(total, GetRewardForBlock(i))
	}

	if total.Cmp(target) != 0 {
		t.Errorf("Total rewards should be %s, got %s", target.String(), total.String())
	}
}
