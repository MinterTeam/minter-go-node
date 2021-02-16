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
	tmjson "github.com/tendermint/tendermint/libs/json"
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
	block, err := s.client.Block(ctx, &height)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Block not found")
	}

	fields := map[pb.BlockField]struct{}{}
	if len(req.Fields) != 0 {
		for _, field := range req.Fields {
			fields[field] = struct{}{}
		}
	} else {
		for _, field := range pb.BlockField_value {
			fields[pb.BlockField(field)] = struct{}{}
		}
	}

	var blockResults *core_types.ResultBlockResults
	if _, ok := fields[pb.BlockField_transactions]; ok {
		blockResults, err = s.client.BlockResults(ctx, &height)
		if err != nil {
			return nil, status.Error(codes.NotFound, "Block results not found")
		}
	}

	var totalValidators []*tmTypes.Validator
	{
		_, okValidators := fields[pb.BlockField_validators]
		_, okEvidence := fields[pb.BlockField_evidence]
		if okValidators || okEvidence {
			valHeight := height - 1
			if valHeight < 1 {
				valHeight = 1
			}

			var page = 1
			var perPage = 100
			tmValidators, err := s.client.Validators(ctx, &valHeight, &page, &perPage)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			totalValidators = tmValidators.Validators
		}
	}

	response := &pb.BlockResponse{
		Hash:             hex.EncodeToString(block.Block.Hash()),
		Height:           uint64(block.Block.Height),
		Time:             block.Block.Time.Format(time.RFC3339Nano),
		TransactionCount: uint64(len(block.Block.Txs)),
	}

	for _, field := range req.Fields {
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}
		switch field {
		case pb.BlockField_size:
			response.Size = uint64(block.Block.Size())
		case pb.BlockField_block_reward:
			response.BlockReward = rewards.GetRewardForBlock(uint64(height)).String()
		case pb.BlockField_transactions:
			response.Transactions, err = s.blockTransaction(block, blockResults, s.blockchain.CurrentState().Coins(), req.FailedTxs)
			if err != nil {
				return nil, err
			}
		case pb.BlockField_proposer:
			response.Proposer, err = blockProposer(block, totalValidators)
			if err != nil {
				return nil, err
			}
		case pb.BlockField_validators:
			response.Validators = blockValidators(totalValidators, block)
		case pb.BlockField_evidence:
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
		data, err := tmjson.Marshal(evidence)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		str, err := encodeToStruct(data)
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
			PublicKey: fmt.Sprintf("Mp%x", tmval.PubKey.Bytes()[:]),
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

func (s *Service) blockTransaction(block *core_types.ResultBlock, blockResults *core_types.ResultBlockResults, coins coins.RCoins, failed bool) ([]*pb.TransactionResponse, error) {
	txs := make([]*pb.TransactionResponse, 0, len(block.Block.Data.Txs))

	for i, rawTx := range block.Block.Data.Txs {
		if blockResults.TxsResults[i].Code != 0 && !failed {
			continue
		}

		tx, _ := transaction.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		tags := make(map[string]string)
		for _, tag := range blockResults.TxsResults[i].Events[0].Attributes {
			key := string(tag.Key)
			value := string(tag.Value)
			tags[key] = value
		}

		data, err := encode(tx.GetDecodedData(), coins)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		txs = append(txs, &pb.TransactionResponse{
			Hash:        strings.Title(fmt.Sprintf("Mt%x", rawTx.Hash())),
			RawTx:       fmt.Sprintf("%x", []byte(rawTx)),
			Height:      uint64(block.Block.Height),
			Index:       uint64(i),
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
			copy(result[:], tmval.PubKey.Bytes()[:])
			return &result
		}
	}

	return nil
}
