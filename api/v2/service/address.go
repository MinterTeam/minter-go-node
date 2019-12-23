package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/types"
)

func (s *Service) Address(_ context.Context, req *pb.AddressRequest) (*pb.AddressResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.AddressResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	address := types.StringToAddress(req.Address)
	response := &pb.AddressResponse{
		Balance:          make(map[string]string),
		TransactionCount: fmt.Sprintf("%d", cState.Accounts.GetNonce(address)),
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
