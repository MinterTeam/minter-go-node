package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func (s *Service) Block(ctx context.Context, req *pb.BlockRequest) (*pb.BlockResponse, error) {
	height := int64(req.Height)
	block, err := s.client.Block(&height)
	if err != nil {
		return new(pb.BlockResponse), status.Error(codes.NotFound, "Block not found")
	}

	blockResults, err := s.client.BlockResults(&height)
	if err != nil {
		return new(pb.BlockResponse), status.Error(codes.NotFound, "Block results not found")
	}

	valHeight := height - 1
	if valHeight < 1 {
		valHeight = 1
	}

	response := &pb.BlockResponse{
		Hash:              hex.EncodeToString(block.Block.Hash()),
		Height:            fmt.Sprintf("%d", block.Block.Height),
		Time:              block.Block.Time.Format(time.RFC3339Nano),
		TransactionsCount: fmt.Sprintf("%d", len(block.Block.Txs)),
	}

	var totalValidators []*tmTypes.Validator

	if len(req.Fields) == 0 {
		response.Size = fmt.Sprintf("%d", len(s.cdc.MustMarshalBinaryLengthPrefixed(block)))
		response.BlockReward = rewards.GetRewardForBlock(uint64(height)).String()

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.BlockResponse), timeoutStatus.Err()
		}

		response.Transactions, err = s.blockTransaction(block, blockResults)
		if err != nil {
			return new(pb.BlockResponse), err
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.BlockResponse), timeoutStatus.Err()
		}

		tmValidators, err := s.client.Validators(&valHeight, 1, 100)
		if err != nil {
			return new(pb.BlockResponse), status.Error(codes.Internal, err.Error())
		}
		totalValidators = tmValidators.Validators

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.BlockResponse), timeoutStatus.Err()
		}

		response.Proposer, err = blockProposer(block, totalValidators)
		if err != nil {
			return new(pb.BlockResponse), err
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.BlockResponse), timeoutStatus.Err()
		}

		response.Validators = blockValidators(totalValidators, block)

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.BlockResponse), timeoutStatus.Err()
		}

		response.Evidence = blockEvidence(block)

		return response, nil
	}

	for _, field := range req.Fields {
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.BlockResponse), timeoutStatus.Err()
		}
		switch field {
		case pb.BlockRequest_size:
			response.Size = fmt.Sprintf("%d", len(s.cdc.MustMarshalBinaryLengthPrefixed(block)))
		case pb.BlockRequest_block_reward:
			response.BlockReward = rewards.GetRewardForBlock(uint64(height)).String()
		case pb.BlockRequest_transactions:
			response.Transactions, err = s.blockTransaction(block, blockResults)
			if err != nil {
				return new(pb.BlockResponse), err
			}
		case pb.BlockRequest_proposer, pb.BlockRequest_validators:
			if len(totalValidators) == 0 {
				tmValidators, err := s.client.Validators(&valHeight, 1, 100)
				if err != nil {
					return new(pb.BlockResponse), status.Error(codes.Internal, err.Error())
				}
				totalValidators = tmValidators.Validators
			}

			if pb.BlockRequest_validators == field {
				response.Validators = blockValidators(totalValidators, block)
				continue
			}

			response.Proposer, err = blockProposer(block, totalValidators)
			if err != nil {
				return new(pb.BlockResponse), err
			}
		case pb.BlockRequest_evidence:
			response.Evidence = blockEvidence(block)
		}

	}

	return response, nil
}

func blockEvidence(block *core_types.ResultBlock) *pb.BlockResponse_Evidence {
	evidences := make([]*pb.BlockResponse_Evidence_Evidence, 0, len(block.Block.Evidence.Evidence))
	for _, evidence := range block.Block.Evidence.Evidence {
		evidences = append(evidences, &pb.BlockResponse_Evidence_Evidence{
			Height:  fmt.Sprintf("%d", evidence.Height()),
			Time:    evidence.Time().Format(time.RFC3339Nano),
			Address: fmt.Sprintf("%s", evidence.Address()),
			Hash:    fmt.Sprintf("%s", evidence.Hash()),
		})
	}
	return &pb.BlockResponse_Evidence{Evidence: evidences}
}

func blockValidators(totalValidators []*tmTypes.Validator, block *core_types.ResultBlock) []*pb.BlockResponse_Validator {
	validators := make([]*pb.BlockResponse_Validator, 0, len(totalValidators))
	for _, tmval := range totalValidators {
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

	return validators
}

func blockProposer(block *core_types.ResultBlock, totalValidators []*tmTypes.Validator) (string, error) {
	p, err := getBlockProposer(block, totalValidators)
	if err != nil {
		return "", status.Error(codes.FailedPrecondition, err.Error())
	}

	if p != nil {
		return p.String(), nil
	}
	return "", nil
}

func (s *Service) blockTransaction(block *core_types.ResultBlock, blockResults *core_types.ResultBlockResults) ([]*pb.BlockResponse_Transaction, error) {
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
			return nil, status.Error(codes.InvalidArgument, err.Error())
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
	return txs, nil
}

func getBlockProposer(block *core_types.ResultBlock, vals []*tmTypes.Validator) (*types.Pubkey, error) {
	for _, tmval := range vals {
		if bytes.Equal(tmval.Address.Bytes(), block.Block.ProposerAddress.Bytes()) {
			var result types.Pubkey
			copy(result[:], tmval.PubKey.Bytes()[5:])
			return &result, nil
		}
	}

	return nil, status.Error(codes.NotFound, "Block proposer not found")
}
