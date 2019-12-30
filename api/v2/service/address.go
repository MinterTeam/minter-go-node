package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) Address(_ context.Context, req *pb.AddressRequest) (*pb.AddressResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.AddressResponse), status.Error(codes.NotFound, err.Error())
	}

	address := types.StringToAddress(req.Address)
	response := &pb.AddressResponse{
		Balance:           make(map[string]string),
		CountTransactions: fmt.Sprintf("%d", cState.Accounts.GetNonce(address)),
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
