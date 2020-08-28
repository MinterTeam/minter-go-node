package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) SendTransaction(ctx context.Context, req *pb.SendTransactionRequest) (*pb.SendTransactionResponse, error) {
	if len(req.Tx) < 3 {
		return new(pb.SendTransactionResponse), status.Error(codes.InvalidArgument, "invalid tx")
	}
	decodeString, err := hex.DecodeString(req.Tx[2:])
	if err != nil {
		return new(pb.SendTransactionResponse), status.Error(codes.InvalidArgument, err.Error())
	}

	result, err := s.broadcastTxSync(decodeString, ctx /*timeout*/)
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return new(pb.SendTransactionResponse), err
		}
		return new(pb.SendTransactionResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	switch result.Code {
	case code.OK:
		return &pb.SendTransactionResponse{
			Code: fmt.Sprintf("%d", result.Code),
			Log:  result.Log,
			Hash: result.Hash.String(),
		}, nil
	default:
		return new(pb.SendTransactionResponse), s.createError(status.New(codes.InvalidArgument, result.Log), result.Info)
	}
}

type ResultBroadcastTx struct {
	Code uint32         `json:"code"`
	Data bytes.HexBytes `json:"data"`
	Log  string         `json:"log"`
	Info string         `json:"-"`
	Hash bytes.HexBytes `json:"hash"`
}

func (s *Service) broadcastTxSync(tx types.Tx, ctx context.Context) (*ResultBroadcastTx, error) {
	resCh := make(chan *abci.Response, 1)
	err := s.tmNode.Mempool().CheckTx(tx, func(res *abci.Response) {
		resCh <- res
	}, mempool.TxInfo{})
	if err != nil {
		return nil, err
	}

	select {
	case res := <-resCh:
		r := res.GetCheckTx()
		return &ResultBroadcastTx{
			Code: r.Code,
			Data: r.Data,
			Log:  r.Log,
			Info: r.Info,
			Hash: tx.Hash(),
		}, nil
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			return nil, status.New(codes.Canceled, ctx.Err().Error()).Err()
		}
		return nil, status.New(codes.DeadlineExceeded, ctx.Err().Error()).Err()
	}

}
