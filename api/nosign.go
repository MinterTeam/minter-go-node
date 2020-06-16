package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
)

type MissBlock struct {
	Height   int64  `json:"height"`
	Proposer string `json:"proposer,omitempty"`
	Pubkey   string `json:"pub_key"`
}

func NoSign(height int64) ([]*MissBlock, error) {
	block, err := client.Block(&height)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Block not found", Data: err.Error()}
	}

	valHeight := height - 1
	if valHeight < 1 {
		valHeight = 1
	}

	var mbs []*MissBlock

	totalValidators, err := client.Validators(&valHeight, 1, 100)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 500, Message: err.Error()}
	}
	var proposer string
	if height <= 1 {
		return mbs, nil
	}
	p, err := getBlockProposer(block, totalValidators.Validators)
	if err != nil {
		return nil, err
	}
	if p != nil {
		proposer = p.String()
	}

	sign := map[string]struct{}{}
	for _, vote := range block.Block.LastCommit.Signatures {
		sign[vote.ValidatorAddress.String()] = struct{}{}
	}

	for _, tmval := range totalValidators.Validators {
		if _, ok := sign[tmval.Address.String()]; ok {
			continue
		}
		mb := &MissBlock{
			Height:   height,
			Proposer: proposer,
			Pubkey:   fmt.Sprintf("Mp%x", tmval.PubKey.Bytes()[5:]),
		}
		mbs = append(mbs, mb)
		break
	}

	return mbs, nil
}
