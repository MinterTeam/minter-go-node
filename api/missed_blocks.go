package api

import (
	"bytes"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
)

type MissedBlocksResponse struct {
	MissedBlocks      *types.BitArray `json:"missed_blocks"`
	MissedBlocksCount int             `json:"missed_blocks_count"`
}

func MissedBlocks(pubkey []byte, height int) (*MissedBlocksResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	vals := cState.GetStateValidators()
	if vals == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Validators not found"}
	}

	for _, val := range vals.Data() {
		if bytes.Equal(val.PubKey, pubkey) {
			return &MissedBlocksResponse{
				MissedBlocks:      val.AbsentTimes,
				MissedBlocksCount: val.CountAbsentTimes(),
			}, nil
		}
	}

	return nil, rpctypes.RPCError{Code: 404, Message: "Validator not found"}
}
