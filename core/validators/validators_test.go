package validators

import (
	"testing"
)

type Results struct {
	Block  uint64
	Result int
}

func TestGetValidatorsCountForBlock(t *testing.T) {
	data := []Results{
		{
			Block:  1,
			Result: 16,
		},
		{
			Block:  518400 * 2,
			Result: 24,
		},
		{
			Block:  31104000,
			Result: 256,
		},
		{
			Block:  31104000 * 2,
			Result: 256,
		},
	}

	for _, item := range data {
		result := GetValidatorsCountForBlock(item.Block)

		if result != item.Result {
			t.Errorf("GetValidatorsCountForBlock result is not correct. Expected %d, got %d", item.Result, result)
		}
	}
}
