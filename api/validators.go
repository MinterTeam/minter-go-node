package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type ValidatorResponse struct {
	Pubkey      types.Pubkey `json:"pub_key"`
	VotingPower int64        `json:"voting_power"`
}

type ResponseValidators []ValidatorResponse

func Validators(height uint64) (*ResponseValidators, error) {
	if height == 0 {
		height = blockchain.Height()
	}

	h := int64(height)
	tmVals, err := client.Validators(&h)
	if err != nil {
		return nil, err
	}

	responseValidators := make(ResponseValidators, len(tmVals.Validators))
	for i, val := range tmVals.Validators {
		responseValidators[i] = ValidatorResponse{
			Pubkey:      val.PubKey.Bytes()[5:],
			VotingPower: val.VotingPower,
		}
	}

	return &responseValidators, nil
}
