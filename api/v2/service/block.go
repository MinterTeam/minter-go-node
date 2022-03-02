package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/minter"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"math/big"
	"strings"
	"time"

	"github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	tmjson "github.com/tendermint/tendermint/libs/json"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Block returns block data at given height.
func (s *Service) Block(ctx context.Context, req *pb.BlockRequest) (*pb.BlockResponse, error) {
	height := int64(req.Height)
	block, err := s.client.Block(ctx, &height)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Block not found")
	}

	fields := map[pb.BlockField]struct{}{}
	if len(req.Fields) > 0 {
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
			return nil, status.Error(codes.NotFound, "Block results not found") // fmt.Sprintf("Block results not found: %v", err))
		}
	}

	var totalValidators []*tmTypes.Validator
	{
		_, okValidators := fields[pb.BlockField_validators]
		_, okProposer := fields[pb.BlockField_proposer]
		if okValidators || okProposer {
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

	if req.Events {
		loadEvents := s.blockchain.GetEventsDB().LoadEvents(uint32(req.Height))
		for _, event := range loadEvents {
			var m proto.Message
			switch e := event.(type) {
			case *events.JailEvent:
				m = &pb.JailEvent{
					ValidatorPubKey: e.ValidatorPubKeyString(),
					JailedUntil:     e.JailedUntil,
				}
			case *events.OrderExpiredEvent:
				m = &pb.OrderExpiredEvent{
					Id:      e.ID,
					Address: e.AddressString(),
					Coin:    e.Coin,
					Amount:  e.Amount,
				}
			case *events.RewardEvent:
				m = &pb.RewardEvent{
					Role:            pb.RewardEvent_Role(pb.RewardEvent_Role_value[e.Role]),
					Address:         e.AddressString(),
					Amount:          e.Amount,
					ForCoin:         e.ForCoin,
					ValidatorPubKey: e.ValidatorPubKeyString(),
				}
			case *events.SlashEvent:
				m = &pb.SlashEvent{
					Address:         e.AddressString(),
					Amount:          e.Amount,
					Coin:            e.Coin,
					ValidatorPubKey: e.ValidatorPubKeyString(),
				}
			case *events.StakeKickEvent:
				m = &pb.StakeKickEvent{
					Address:         e.AddressString(),
					Amount:          e.Amount,
					Coin:            e.Coin,
					ValidatorPubKey: e.ValidatorPubKeyString(),
				}
			case *events.UnbondEvent:
				m = &pb.UnbondEvent{
					Address:         e.AddressString(),
					Amount:          e.Amount,
					Coin:            e.Coin,
					ValidatorPubKey: e.ValidatorPubKeyString(),
				}
			case *events.UnlockEvent:
				m = &pb.UnlockEvent{
					Address: e.AddressString(),
					Amount:  e.Amount,
					Coin:    e.Coin,
				}
			case *events.StakeMoveEvent:
				m = &pb.StakeMoveEvent{
					Address:           e.AddressString(),
					Amount:            e.Amount,
					Coin:              e.Coin,
					CandidatePubKey:   e.CandidatePubKey.String(),
					ToCandidatePubKey: e.ToCandidatePubKey.String(),
				}
			case *events.RemoveCandidateEvent:
				m = &pb.RemoveCandidateEvent{
					CandidatePubKey: e.CandidatePubKeyString(),
				}
			case *events.UpdateNetworkEvent:
				m = &pb.UpdateNetworkEvent{
					Version: e.Version,
				}
			case *events.UpdatedBlockRewardEvent:
				m = &pb.UpdatedBlockRewardEvent{
					Value:                   e.Value,
					ValueLockedStakeRewards: e.ValueLockedStakeRewards,
				}
			case *events.UpdateCommissionsEvent:
				m = &pb.UpdateCommissionsEvent{
					Coin:                    e.Coin,
					PayloadByte:             e.PayloadByte,
					Send:                    e.Send,
					BuyBancor:               e.BuyBancor,
					SellBancor:              e.SellBancor,
					SellAllBancor:           e.SellAllBancor,
					BuyPoolBase:             e.BuyPoolBase,
					BuyPoolDelta:            e.BuyPoolDelta,
					SellPoolBase:            e.SellPoolBase,
					SellPoolDelta:           e.SellPoolDelta,
					SellAllPoolBase:         e.SellAllPoolBase,
					SellAllPoolDelta:        e.SellAllPoolDelta,
					CreateTicker3:           e.CreateTicker3,
					CreateTicker4:           e.CreateTicker4,
					CreateTicker5:           e.CreateTicker5,
					CreateTicker6:           e.CreateTicker6,
					CreateTicker7_10:        e.CreateTicker7_10,
					CreateCoin:              e.CreateCoin,
					CreateToken:             e.CreateToken,
					RecreateCoin:            e.RecreateCoin,
					RecreateToken:           e.RecreateToken,
					DeclareCandidacy:        e.DeclareCandidacy,
					Delegate:                e.Delegate,
					Unbond:                  e.Unbond,
					RedeemCheck:             e.RedeemCheck,
					SetCandidateOn:          e.SetCandidateOn,
					SetCandidateOff:         e.SetCandidateOff,
					CreateMultisig:          e.CreateMultisig,
					MultisendBase:           e.MultisendBase,
					MultisendDelta:          e.MultisendDelta,
					EditCandidate:           e.EditCandidate,
					SetHaltBlock:            e.SetHaltBlock,
					EditTickerOwner:         e.EditTickerOwner,
					EditMultisig:            e.EditMultisig,
					EditCandidatePublicKey:  e.EditCandidatePublicKey,
					CreateSwapPool:          e.CreateSwapPool,
					AddLiquidity:            e.AddLiquidity,
					RemoveLiquidity:         e.RemoveLiquidity,
					EditCandidateCommission: e.EditCandidateCommission,
					MintToken:               e.MintToken,
					BurnToken:               e.BurnToken,
					VoteCommission:          e.VoteCommission,
					VoteUpdate:              e.VoteUpdate,
					FailedTx:                e.FailedTx,
					AddLimitOrder:           e.AddLimitOrder,
					RemoveLimitOrder:        e.RemoveLimitOrder,
				}
			default:
				return nil, status.Error(codes.Internal, "unknown event type")
			}

			a, err := anypb.New(m)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			response.Events = append(response.Events, a)
		}
	}

	for field := range fields {
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}
		switch field {
		case pb.BlockField_size:
			response.Size = uint64(block.Block.Size())
		case pb.BlockField_block_reward:
			if h := s.blockchain.GetVersionHeight(minter.V3); req.Height < h {
				response.BlockReward = wrapperspb.String(s.rewards.GetRewardForBlock(uint64(height)).String())
				continue
			}

			state, err := s.blockchain.GetStateForHeight(req.Height)
			if err != nil { // is ok
				//return nil, status.Error(codes.NotFound, err.Error())
				continue
			}

			reward, rewardWithLock := state.App().Reward()
			response.BlockReward = wrapperspb.String(reward.String())
			response.LockedStakeRewards = wrapperspb.String(new(big.Int).Mul(rewardWithLock, big.NewInt(3)).String())
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

		tx, _ := s.decoderTx.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		tags := make(map[string]string)
		for _, tag := range blockResults.TxsResults[i].Events[0].Attributes {
			key := string(tag.Key)
			value := string(tag.Value)
			tags[key] = value
		}

		data, err := encode(tx.GetDecodedData(), tx.Type, coins)
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
			TypeHex:     tx.Type.String(),
			Type:        tx.Type.UInt64(),
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
