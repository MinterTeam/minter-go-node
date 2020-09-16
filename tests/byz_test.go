package tests

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	tmTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"testing"
	"time"
)

func TestBlockchain_ByzantineValidators(t *testing.T) {
	state := DefaultAppState() // generate default state

	state.Validators = append(state.Validators, types.Validator{
		TotalBipStake: "10000",
		PubKey:        types.Pubkey{1},
		AccumReward:   "10",
		AbsentTimes:   nil,
	})
	state.Candidates = append(state.Candidates, types.Candidate{
		ID:             1,
		RewardAddress:  types.Address{},
		OwnerAddress:   types.Address{},
		ControlAddress: types.Address{},
		TotalBipStake:  "10000",
		PubKey:         types.Pubkey{1},
		Commission:     10,
		Stakes: []types.Stake{
			{
				Owner:    types.Address{},
				Coin:     0,
				Value:    "10000",
				BipValue: "10000",
			},
		},
		Updates: nil,
		Status:  2,
	})

	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], types.Pubkey{1}.Bytes())
	var address types.TmAddress
	copy(address[:], pubkey.Address().Bytes())

	app := CreateApp(state) // create application
	req := tmTypes.RequestBeginBlock{
		Hash: nil,
		Header: tmTypes.Header{
			Version:            tmTypes.Version{},
			ChainID:            "",
			Height:             1,
			Time:               time.Time{},
			LastBlockId:        tmTypes.BlockID{},
			LastCommitHash:     nil,
			DataHash:           nil,
			ValidatorsHash:     nil,
			NextValidatorsHash: nil,
			ConsensusHash:      nil,
			AppHash:            nil,
			LastResultsHash:    nil,
			EvidenceHash:       nil,
			ProposerAddress:    nil,
		},
		LastCommitInfo: tmTypes.LastCommitInfo{
			Round: 0,
			Votes: nil,
		},
		ByzantineValidators: []tmTypes.Evidence{
			{
				Type: "",
				Validator: tmTypes.Validator{
					Address: address[:],
					Power:   10,
				},
				Height:           1,
				Time:             time.Time{},
				TotalVotingPower: 0,
			},
		},
	}
	app.BeginBlock(req)
	// SendBeginBlock(app) // send BeginBlock

	SendEndBlock(app) // send EndBlock
	SendCommit(app)   // send Commit

	if validator := app.CurrentState().Validators().GetByPublicKey([32]byte{1}); validator != nil {
		t.Error("validator exists")
	}
	candidate := app.CurrentState().Candidates().GetCandidate([32]byte{1})
	if candidate == nil {
		t.Fatal("candidate is not exists")
	}
	if candidate.GetTotalBipStake().String() != "0" {
		t.Fatal("candidate total bip stake is not 0")
	}

}
