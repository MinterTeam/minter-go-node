package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) Address(_ context.Context, req *pb.AddressRequest) (*pb.AddressResponse, error) {
	if len(req.Address) < 3 {
		return new(pb.AddressResponse), status.Error(codes.InvalidArgument, "invalid address")
	}

	decodeString, err := hex.DecodeString(req.Address[2:])
	if err != nil {
		return new(pb.AddressResponse), status.Error(codes.InvalidArgument, err.Error())
	}

	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.AddressResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	address := types.BytesToAddress(decodeString)
	response := &pb.AddressResponse{
		Balance:           make(map[string]string),
		TransactionsCount: fmt.Sprintf("%d", cState.Accounts.GetNonce(address)),
	}

	balances := cState.Accounts.GetBalances(address)

	for k, v := range balances {
		response.Balance[k.String()] = v.String()
	}

	if _, exists := response.Balance[types.GetBaseCoin().String()]; !exists {
		response.Balance[types.GetBaseCoin().String()] = "0"
	}

	return response, nil
}
