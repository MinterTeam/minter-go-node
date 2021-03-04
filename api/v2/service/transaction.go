package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"strings"
)

// Transaction returns transaction info.
func (s *Service) Transaction(ctx context.Context, req *pb.TransactionRequest) (*pb.TransactionResponse, error) {
	if len(req.Hash) < 3 {
		return nil, status.Error(codes.InvalidArgument, "invalid hash")
	}
	decodeString, err := hex.DecodeString(req.Hash[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	tx, err := s.client.Tx(ctx, decodeString, false)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	decodedTx, _ := transaction.DecodeFromBytes(tx.Tx)
	sender, _ := decodedTx.Sender()

	tags := make(map[string]string)
	var gas int
	for _, tag := range tx.TxResult.Events[0].Attributes {
		key := string(tag.Key)
		value := string(tag.Value)
		tags[key] = value
		if key == "tx.gas" {
			gas, _ = strconv.Atoi(value)
		}
	}

	cState := s.blockchain.CurrentState()

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	dataStruct, err := encode(decodedTx.GetDecodedData(), cState.Coins())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.TransactionResponse{
		Hash:     "Mt" + strings.ToLower(hex.EncodeToString(tx.Tx.Hash())),
		RawTx:    fmt.Sprintf("%x", []byte(tx.Tx)),
		Height:   uint64(tx.Height),
		Index:    uint64(tx.Index),
		From:     sender.String(),
		Nonce:    decodedTx.Nonce,
		GasPrice: uint64(decodedTx.GasPrice),
		GasCoin: &pb.Coin{
			Id:     uint64(decodedTx.GasCoin),
			Symbol: cState.Coins().GetCoin(decodedTx.GasCoin).GetFullSymbol(),
		},
		Gas:         uint64(gas),
		Type:        uint64(decodedTx.Type),
		Data:        dataStruct,
		Payload:     decodedTx.Payload,
		ServiceData: decodedTx.ServiceData,
		Tags:        tags,
		Code:        uint64(tx.TxResult.Code),
		Log:         tx.TxResult.Log,
	}, nil
}
