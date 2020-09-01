package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
)

func (s *Service) EstimateTxCommission(ctx context.Context, req *pb.EstimateTxCommissionRequest) (*pb.EstimateTxCommissionResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	if req.GetTx() == "" {
		data := req.GetData()
		if data == nil {
			return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "invalid tx and data")
		}

		return commissionCoinForData(data, cState)
	}

	if !strings.HasPrefix(strings.Title(req.GetTx()), "0x") {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "invalid transaction")
	}

	decodeString, err := hex.DecodeString(req.GetTx()[2:])
	if err != nil {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, err.Error())
	}

	decodedTx, err := transaction.TxDecoder.DecodeFromBytesWithoutSig(decodeString)
	if err != nil {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "Cannot decode transaction")
	}

	commissionInBaseCoin := decodedTx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !decodedTx.GasCoin.IsBaseCoin() {
		coin := cState.Coins().GetCoin(decodedTx.GasCoin)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return new(pb.EstimateTxCommissionResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())), transaction.EncodeError(map[string]string{
				"commission_in_base_coin": commissionInBaseCoin.String(),
				"value_has":               coin.Reserve().String(),
				"value_required":          commissionInBaseCoin.String(),
			}))
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	return &pb.EstimateTxCommissionResponse{
		Commission: commission.String(),
	}, nil
}

func commissionCoinForData(data *pb.EstimateTxCommissionRequest_TransactionData, cState *state.CheckState) (*pb.EstimateTxCommissionResponse, error) {
	var commissionInBaseCoin *big.Int
	switch data.Type {
	case pb.EstimateTxCommissionRequest_TransactionData_Send:
		commissionInBaseCoin = big.NewInt(commissions.SendTx)
	case pb.EstimateTxCommissionRequest_TransactionData_SellAllCoin,
		pb.EstimateTxCommissionRequest_TransactionData_SellCoin,
		pb.EstimateTxCommissionRequest_TransactionData_BuyCoin:
		commissionInBaseCoin = big.NewInt(commissions.ConvertTx)
	case pb.EstimateTxCommissionRequest_TransactionData_DeclareCandidacy:
		commissionInBaseCoin = big.NewInt(commissions.DeclareCandidacyTx)
	case pb.EstimateTxCommissionRequest_TransactionData_Delegate:
		commissionInBaseCoin = big.NewInt(commissions.DelegateTx)
	case pb.EstimateTxCommissionRequest_TransactionData_Unbond:
		commissionInBaseCoin = big.NewInt(commissions.UnbondTx)
	case pb.EstimateTxCommissionRequest_TransactionData_SetCandidateOffline,
		pb.EstimateTxCommissionRequest_TransactionData_SetCandidateOnline:
		commissionInBaseCoin = big.NewInt(commissions.ToggleCandidateStatus)
	case pb.EstimateTxCommissionRequest_TransactionData_EditCandidate:
		commissionInBaseCoin = big.NewInt(commissions.EditCandidate)
	case pb.EstimateTxCommissionRequest_TransactionData_RedeemCheck:
		commissionInBaseCoin = big.NewInt(commissions.RedeemCheckTx)
	case pb.EstimateTxCommissionRequest_TransactionData_CreateMultisig:
		commissionInBaseCoin = big.NewInt(commissions.CreateMultisig)
	case pb.EstimateTxCommissionRequest_TransactionData_Multisend:
		if data.Mtxs <= 0 {
			return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "Set number of transactions for multisend (mtxs)")
		}
		commissionInBaseCoin = big.NewInt(commissions.MultisendDelta*(data.Mtxs-1) + 10)
	default:
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "Set correct transaction type for tx")
	}

	lenPayload := len(data.Payload)
	if lenPayload > 1024 {
		return new(pb.EstimateTxCommissionResponse), status.Errorf(codes.InvalidArgument, "Transaction payload length is over %d bytes", 1024)
	}

	totalCommissionInBaseCoin := new(big.Int).Mul(big.NewInt(0).Add(commissionInBaseCoin, big.NewInt(int64(lenPayload))), transaction.CommissionMultiplier)

	if types.CoinID(data.GasCoinId).IsBaseCoin() {
		return &pb.EstimateTxCommissionResponse{
			Commission: totalCommissionInBaseCoin.String(),
		}, nil
	}

	coin := cState.Coins().GetCoin(types.CoinID(data.GasCoinId))

	if coin == nil {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "Gas Coin not found")
	}

	if totalCommissionInBaseCoin.Cmp(coin.Reserve()) == 1 {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "Not enough coin reserve for pay comission")
	}

	return &pb.EstimateTxCommissionResponse{
		Commission: formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), totalCommissionInBaseCoin).String(),
	}, nil

}
