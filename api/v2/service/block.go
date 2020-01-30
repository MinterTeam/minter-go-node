package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func (s *Service) Block(_ context.Context, req *pb.BlockRequest) (*pb.BlockResponse, error) {
	block, err := s.client.Block(&req.Height)
	if err != nil {
		return new(pb.BlockResponse), status.Error(codes.NotFound, "Block not found")
	}

	blockResults, err := s.client.BlockResults(&req.Height)
	if err != nil {
		return new(pb.BlockResponse), status.Error(codes.NotFound, "Block results not found")
	}

	valHeight := req.Height - 1
	if valHeight < 1 {
		valHeight = 1
	}

	tmValidators, err := s.client.Validators(&valHeight, 1, 256)
	if err != nil {
		return new(pb.BlockResponse), status.Error(codes.NotFound, "Validators for block not found")
	}

	txs := make([]*pb.BlockResponse_Transaction, 0, len(block.Block.Data.Txs))
	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.TxDecoder.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		tags := make(map[string]string)
		for _, tag := range blockResults.TxsResults[i].Events[0].Attributes {
			tags[string(tag.Key)] = string(tag.Value)
		}

		dataStruct, err := s.encodeTxData(tx)
		if err != nil {
			return new(pb.BlockResponse), status.Error(codes.InvalidArgument, err.Error())
		}

		txs = append(txs, &pb.BlockResponse_Transaction{
			Hash:        fmt.Sprintf("Mt%x", rawTx.Hash()),
			RawTx:       fmt.Sprintf("%x", []byte(rawTx)),
			From:        sender.String(),
			Nonce:       fmt.Sprintf("%d", tx.Nonce),
			GasPrice:    fmt.Sprintf("%d", tx.GasPrice),
			Type:        fmt.Sprintf("%d", tx.Type),
			Data:        dataStruct,
			Payload:     tx.Payload,
			ServiceData: tx.ServiceData,
			Gas:         fmt.Sprintf("%d", tx.Gas()),
			GasCoin:     tx.GasCoin.String(),
			Tags:        tags,
			Code:        fmt.Sprintf("%d", blockResults.TxsResults[i].Code),
			Log:         blockResults.TxsResults[i].Log,
		})
	}

	var validators []*pb.BlockResponse_Validator
	var proposer string
	if req.Height > 1 {
		p, err := s.getBlockProposer(block)
		if err != nil {
			return new(pb.BlockResponse), status.Error(codes.FailedPrecondition, err.Error())
		}

		if p != nil {
			str := p.String()
			proposer = str
		}

		validators = make([]*pb.BlockResponse_Validator, 0, len(tmValidators.Validators))
		for _, tmval := range tmValidators.Validators {
			signed := false
			for _, vote := range block.Block.LastCommit.Signatures {
				if bytes.Equal(vote.ValidatorAddress.Bytes(), tmval.Address.Bytes()) {
					signed = true
					break
				}
			}

			validators = append(validators, &pb.BlockResponse_Validator{
				PublicKey: fmt.Sprintf("Mp%x", tmval.PubKey.Bytes()[5:]),
				Signed:    signed,
			})
		}
	}

	evidences := make([]*pb.BlockResponse_Evidence_Evidence, len(block.Block.Evidence.Evidence))
	for _, evidence := range block.Block.Evidence.Evidence {
		evidences = append(evidences, &pb.BlockResponse_Evidence_Evidence{
			Height:  fmt.Sprintf("%d", evidence.Height()),
			Time:    evidence.Time().Format(time.RFC3339Nano),
			Address: fmt.Sprintf("%s", evidence.Address()),
			Hash:    fmt.Sprintf("%s", evidence.Hash()),
		})
	}
	return &pb.BlockResponse{
		Hash:              hex.EncodeToString(block.Block.Hash()),
		Height:            fmt.Sprintf("%d", block.Block.Height),
		Time:              block.Block.Time.Format(time.RFC3339Nano),
		TotalTransactions: fmt.Sprintf("%d", len(block.Block.Txs)),
		Transactions:      txs,
		BlockReward:       rewards.GetRewardForBlock(uint64(req.Height)).String(),
		Size:              fmt.Sprintf("%d", s.cdc.MustMarshalBinaryLengthPrefixed(block)),
		Proposer:          proposer,
		Validators:        validators,
		Evidence: &pb.BlockResponse_Evidence{
			Evidence: evidences, // todo
		},
	}, nil
}

func (s *Service) getBlockProposer(block *core_types.ResultBlock) (*types.Pubkey, error) {
	vals, err := s.client.Validators(&block.Block.Height, 1, 256)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Validators for block not found")
	}

	for _, tmval := range vals.Validators {
		if bytes.Equal(tmval.Address.Bytes(), block.Block.ProposerAddress.Bytes()) {
			var result types.Pubkey
			copy(result[:], tmval.PubKey.Bytes()[5:])
			return &result, nil
		}
	}

	return nil, status.Error(codes.NotFound, "Block proposer not found")
}
