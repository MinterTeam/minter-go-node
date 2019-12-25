package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/code"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) SendTransaction(_ context.Context, req *pb.SendTransactionRequest) (*pb.SendTransactionResponse, error) {
	decodeString, err := hex.DecodeString(req.Tx[2:])
	if err != nil {
		return new(pb.SendTransactionResponse), s.createError(status.New(codes.InvalidArgument, err.Error()), nil)
	}
	result, err := s.client.BroadcastTxSync(decodeString)
	if err != nil {
		return new(pb.SendTransactionResponse), s.createError(status.New(codes.FailedPrecondition, err.Error()), nil)
	}

	switch result.Code {
	// general
	case code.WrongNonce:
	case code.CoinNotExists:
	case code.CoinReserveNotSufficient:
	case code.TxTooLarge:
	case code.DecodeError:
	case code.InsufficientFunds:
	case code.TxPayloadTooLarge:
	case code.TxServiceDataTooLarge:
	case code.InvalidMultisendData:
	case code.CoinSupplyOverflow:
	case code.TxFromSenderAlreadyInMempool:
	case code.TooLowGasPrice:
	case code.WrongChainID:
	case code.CoinReserveUnderflow:

		// coin creation
	case code.CoinAlreadyExists:
	case code.WrongCrr:
	case code.InvalidCoinSymbol:
	case code.InvalidCoinName:
	case code.WrongCoinSupply:

		// convert
	case code.CrossConvert:
	case code.MaximumValueToSellReached:
	case code.MinimumValueToBuyReached:

		// candidate
	case code.CandidateExists:
	case code.WrongCommission:
	case code.CandidateNotFound:
	case code.StakeNotFound:
	case code.InsufficientStake:
	case code.IsNotOwnerOfCandidate:
	case code.IncorrectPubKey:
	case code.StakeShouldBePositive:
	case code.TooLowStake:

		// check
	case code.CheckInvalidLock:
	case code.CheckExpired:
	case code.CheckUsed:
	case code.TooHighGasPrice:
	case code.WrongGasCoin:
	case code.TooLongNonce:

		// multisig
	case code.IncorrectWeights:
	case code.MultisigExists:
	case code.MultisigNotExists:
	case code.IncorrectMultiSignature:
	case code.TooLargeOwnersList:

		//OK
	case code.OK:
		fallthrough
	default:
		return &pb.SendTransactionResponse{
			Code: fmt.Sprintf("%d", result.Code),
			Log:  result.Log,
			Hash: result.Hash.String(),
		}, nil
	}

	return new(pb.SendTransactionResponse), s.createError(status.New(codes.InvalidArgument, result.Log), result.Data)
}
