package tests_old

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	tmTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmTypes1 "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proto/tendermint/version"
	"math/big"
	"testing"
	"time"
)

func TestBlockchain_ByzantineValidators(t *testing.T) {
	state := DefaultAppState() // generate default state

	stake := helpers.BipToPip(big.NewInt(1000)).String()
	state.Validators = append(state.Validators, types.Validator{
		TotalBipStake: stake,
		PubKey:        types.Pubkey{1},
		AccumReward:   "10",
		AbsentTimes:   nil,
	})
	state.Candidates = append(state.Candidates, types.Candidate{
		ID:             1,
		RewardAddress:  types.Address{},
		OwnerAddress:   types.Address{},
		ControlAddress: types.Address{},
		TotalBipStake:  stake,
		PubKey:         types.Pubkey{1},
		Commission:     10,
		Stakes: []types.Stake{
			{
				Owner:    types.Address{},
				Coin:     0,
				Value:    stake,
				BipValue: stake,
			},
		},
		Updates: nil,
		Status:  2,
	})

	var address types.TmAddress
	bytes := [32]byte{1}
	copy(address[:], ed25519.PubKey(bytes[:]).Address().Bytes())

	app := CreateApp(state) // create application
	req := tmTypes.RequestBeginBlock{
		Hash: nil,
		Header: tmTypes1.Header{
			Version:            version.Consensus{},
			ChainID:            "",
			Height:             1,
			Time:               time.Time{},
			LastBlockId:        tmTypes1.BlockID{},
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
				Type: tmTypes.EvidenceType_DUPLICATE_VOTE,
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

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

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
