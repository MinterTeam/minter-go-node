package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/code"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

func (s *Service) SendTransaction(_ context.Context, req *pb.SendTransactionRequest) (*pb.SendTransactionResponse, error) {
	result, err := s.client.BroadcastTxSync([]byte(req.Tx))
	if err != nil {
		return new(pb.SendTransactionResponse), s.createError(status.New(codes.FailedPrecondition, err.Error()), nil)
	}

	var details = make(map[string]string)
	fields := strings.Fields(strings.ReplaceAll(result.Log, ",", ""))

	switch result.Code {
	// general
	case code.WrongNonce:
		details["description"] = "wrong_nonce"
		details["expected"] = fields[3]
		details["got"] = fields[5]
	case code.CoinNotExists:
		details["description"] = "coin_not_exists"
		details["coin"] = fields[1]
	case code.CoinReserveNotSufficient:
	case code.TxTooLarge:
		details["description"] = "tx_too_large"
		details["max_tx_length"] = fields[4]
	case code.DecodeError:
		details["description"] = "decode_error"
	case code.InsufficientFunds:
		details["description"] = "insufficient_funds"
		details["sender"] = fields[5]
		details["value"] = fields[7]
		details["coin"] = fields[8]
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
			Data: result.Data.String(),
			Log:  result.Log,
			Hash: result.Hash.String(),
		}, nil
	}

	details["code"] = fmt.Sprintf("%d", result.Code)
	return new(pb.SendTransactionResponse), s.createError(status.New(codes.InvalidArgument, result.Log), details)
}
