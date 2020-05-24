package api

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	tmTypes "github.com/tendermint/tendermint/types"
)

type MissBlock struct {
	Height       int64      `json:"height"`
	Proposer     *string   	`json:"proposer,omitempty"`
	Pubkey 	     string 	`json:"pub_key"`
	Signed       bool   	`json:"signed"`
}

func NoSign(height int64) ([]MissBlock, error) {
	block, err := client.Block(&height)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Block not found", Data: err.Error()}
	}

	valHeight := height - 1
	if valHeight < 1 {
		valHeight = 1
	}
	
	var mbs []MissBlock
	var mb MissBlock 

	var totalValidators []*tmTypes.Validator
	for i := 0; i < (((len(block.Block.LastCommit.Signatures) - 1) / 100) + 1); i++ {
		tmValidators, err := client.Validators(&valHeight, i+1, 100)
		if err != nil {
			return nil, rpctypes.RPCError{Code: 500, Message: err.Error()}
		}
		totalValidators = append(totalValidators, tmValidators.Validators...)
	}
	var proposer *string
	if height > 1 {
		p, err := getBlockProposer(block, totalValidators)
		if err != nil {
			return nil, err
		}

		if p != nil {
			str := p.String()
			proposer = &str
		}

		for _, tmval := range totalValidators {
			signed := false
			for _, vote := range block.Block.LastCommit.Signatures {
				if bytes.Equal(vote.ValidatorAddress.Bytes(), tmval.Address.Bytes()) {
					signed = true
					break
				}
			}

			if signed == false {
			mb.Height = height 
			mb.Proposer = proposer 
			mb.Pubkey = fmt.Sprintf("Mp%x", tmval.PubKey.Bytes()[5:])
			mb.Signed = signed
			mbs = append(mbs,mb)
			}
		}
	}

	return mbs, nil
}

