package rewards

import (
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"sync"
	"testing"
)

type Results struct {
	Block  uint64
	Result *big.Int
}

func NewFrom0Block() *Reward {
	return &Reward{startHeight: 0}
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
		result := NewFrom0Block().GetRewardForBlock(item.Block)

		if result.Cmp(item.Result) != 0 {
			t.Errorf("GetRewardForBlock result is not correct. Expected %s, got %s", item.Result.String(), result.String())
		}
	}
}

func TestTotalRewardsCount(t *testing.T) {
	target := helpers.BipToPip(big.NewInt(9800000000))
	fromHeight := uint64(1)
	toHeight := uint64(43703000)
	const routines = 4
	results := [routines]*big.Int{}
	chunk := (toHeight - fromHeight) / routines

	wg := sync.WaitGroup{}
	wg.Add(routines)

	for i := uint64(0); i < routines; i++ {
		go func(i uint64) {
			results[i] = big.NewInt(0)
			for block := fromHeight + (i * chunk); block < fromHeight+((i+1)*chunk) && block < toHeight; block++ {
				results[i].Add(results[i], NewFrom0Block().GetRewardForBlock(block))
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	total := big.NewInt(0)
	for _, result := range results {
		total.Add(total, result)
	}

	if total.Cmp(target) != 0 {
		t.Errorf("Total rewards should be %s, got %s", target.String(), total.String())
	}
}
