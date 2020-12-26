package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/state/coins"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

// Block returns block data at given height.
func (s *Service) Block(ctx context.Context, req *pb.BlockRequest) (*pb.BlockResponse, error) {
	height := int64(req.Height)
	block, err := s.client.Block(&height)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Block not found")
	}

	blockResults, err := s.client.BlockResults(&height)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Block results not found")
	}

	valHeight := height - 1
	if valHeight < 1 {
		valHeight = 1
	}

	response := &pb.BlockResponse{
		Hash:             hex.EncodeToString(block.Block.Hash()),
		Height:           uint64(block.Block.Height),
		Time:             block.Block.Time.Format(time.RFC3339Nano),
		TransactionCount: uint64(len(block.Block.Txs)),
	}

	var totalValidators []*tmTypes.Validator

	if len(req.Fields) == 0 {
		response.Size = uint64(len(s.cdc.MustMarshalBinaryLengthPrefixed(block)))
		response.BlockReward = rewards.GetRewardForBlock(uint64(height)).String()

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		currentState := s.blockchain.CurrentState()

		response.Transactions, err = s.blockTransaction(block, blockResults, currentState.Coins())
		if err != nil {
			return nil, err
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		tmValidators, err := s.client.Validators(&valHeight, 1, 100)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		totalValidators = tmValidators.Validators

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		response.Proposer, err = blockProposer(block, totalValidators)
		if err != nil {
			return nil, err
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		response.Validators = blockValidators(totalValidators, block)

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		response.Evidence, err = blockEvidence(block)
		if err != nil {
			return nil, err
		}

		return response, nil
	}

	for _, field := range req.Fields {
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}
		switch field {
		case pb.BlockRequest_size:
			response.Size = uint64(len(s.cdc.MustMarshalBinaryLengthPrefixed(block)))
		case pb.BlockRequest_block_reward:
			response.BlockReward = rewards.GetRewardForBlock(uint64(height)).String()
		case pb.BlockRequest_transactions:
			cState := s.blockchain.CurrentState()

			response.Transactions, err = s.blockTransaction(block, blockResults, cState.Coins())
			if err != nil {
				return nil, err
			}
		case pb.BlockRequest_proposer, pb.BlockRequest_validators:
			if len(totalValidators) == 0 {
				tmValidators, err := s.client.Validators(&valHeight, 1, 100)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
				totalValidators = tmValidators.Validators
			}

			if pb.BlockRequest_validators == field {
				response.Validators = blockValidators(totalValidators, block)
				continue
			}

			response.Proposer, err = blockProposer(block, totalValidators)
			if err != nil {
				return nil, err
			}
		case pb.BlockRequest_evidence:
			response.Evidence, err = blockEvidence(block)
			if err != nil {
				return nil, err
			}
		}

	}

	return response, nil
}

func blockEvidence(block *core_types.ResultBlock) (*pb.BlockResponse_Evidence, error) {
	evidences := make([]*_struct.Struct, 0, len(block.Block.Evidence.Evidence))
	for _, evidence := range block.Block.Evidence.Evidence {
		proto, err := tmTypes.EvidenceToProto(evidence)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		str, err := toStruct(proto.GetSum())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		evidences = append(evidences, str)
	}
	return &pb.BlockResponse_Evidence{Evidence: evidences}, nil
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
	p := getBlockProposer(block, totalValidators)
	if p != nil {
		return p.String(), nil
	}
	return "", nil
}

func (s *Service) blockTransaction(block *core_types.ResultBlock, blockResults *core_types.ResultBlockResults, coins coins.RCoins) ([]*pb.BlockResponse_Transaction, error) {
	txs := make([]*pb.BlockResponse_Transaction, 0, len(block.Block.Data.Txs))

	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		tags := make(map[string]string)
		for _, tag := range blockResults.TxsResults[i].Events[0].Attributes {
			tags[string(tag.Key)] = string(tag.Value)
		}

		data, err := encode(tx.GetDecodedData(), coins)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		txs = append(txs, &pb.BlockResponse_Transaction{
			Hash:        strings.Title(fmt.Sprintf("Mt%x", rawTx.Hash())),
			RawTx:       fmt.Sprintf("%x", []byte(rawTx)),
			From:        sender.String(),
			Nonce:       tx.Nonce,
			GasPrice:    uint64(tx.GasPrice),
			Type:        uint64(tx.Type),
			Data:        data,
			Payload:     tx.Payload,
			ServiceData: tx.ServiceData,
			Gas:         uint64(tx.Gas()),
			GasCoin: &pb.Coin{
				Id:     uint64(tx.GasCoin),
				Symbol: coins.GetCoin(tx.GasCoin).GetFullSymbol(),
			},
			Tags: tags,
			Code: uint64(blockResults.TxsResults[i].Code),
			Log:  blockResults.TxsResults[i].Log,
		})
	}
	return txs, nil
}

func getBlockProposer(block *core_types.ResultBlock, vals []*tmTypes.Validator) *types.Pubkey {
	for _, tmval := range vals {
		if bytes.Equal(tmval.Address.Bytes(), block.Block.ProposerAddress.Bytes()) {
			var result types.Pubkey
			copy(result[:], tmval.PubKey.Bytes()[5:])
			return &result
		}
	}

	return nil
}
