package service

import (
	"context"
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/core/code"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

// SendTransaction returns the result of sending signed tx. To ensure that transaction was successfully committed to the blockchain, you need to find the transaction by the hash and ensure that the status code equals to 0.
func (s *Service) SendTransaction(ctx context.Context, req *pb.SendTransactionRequest) (*pb.SendTransactionResponse, error) {
	if !strings.HasPrefix(strings.Title(req.GetTx()), "0x") {
		return nil, status.Error(codes.InvalidArgument, "invalid transaction")
	}
	decodeString, err := hex.DecodeString(req.Tx[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	result, statusErr := s.broadcastTxSync(decodeString, ctx /*timeout*/)
	if statusErr != nil {
		return new(pb.SendTransactionResponse), statusErr.Err()
	}

	switch result.Code {
	case code.OK:
		return &pb.SendTransactionResponse{
			Code: uint64(result.Code),
			Log:  result.Log,
			Hash: "Mt" + strings.ToLower(result.Hash.String()),
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

func (s *Service) broadcastTxSync(tx types.Tx, ctx context.Context) (*ResultBroadcastTx, *status.Status) {
	resCh := make(chan *abci.Response, 1)
	err := s.tmNode.Mempool().CheckTx(tx, func(res *abci.Response) {
		resCh <- res
	}, mempool.TxInfo{})
	if err != nil {
		if err.Error() == mempool.ErrTxInCache.Error() {
			return nil, status.New(codes.AlreadyExists, err.Error())
		}
		return nil, status.New(codes.FailedPrecondition, err.Error())
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
			return nil, status.New(codes.Canceled, ctx.Err().Error())
		}
		return nil, status.New(codes.DeadlineExceeded, ctx.Err().Error())
	}

}
