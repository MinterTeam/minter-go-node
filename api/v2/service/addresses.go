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

func (s *Service) Addresses(_ context.Context, req *pb.AddressesRequest) (*pb.AddressesResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.AddressesResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	response := &pb.AddressesResponse{
		Addresses: make([]*pb.AddressesResponse_Result, 0, len(req.Addresses)),
	}

	for _, address := range req.Addresses {
		if len(address) < 3 {
			return new(pb.AddressesResponse), status.Error(codes.InvalidArgument, fmt.Sprintf("invalid address %s", address))
		}

		decodeString, err := hex.DecodeString(address[2:])
		if err != nil {
			return new(pb.AddressesResponse), status.Error(codes.InvalidArgument, err.Error())
		}
		addr := types.BytesToAddress(decodeString)
		data := &pb.AddressesResponse_Result{
			Address:           address,
			Balance:           make(map[string]string),
			TransactionsCount: fmt.Sprintf("%d", cState.Accounts.GetNonce(addr)),
		}

		balances := cState.Accounts.GetBalances(addr)
		for k, v := range balances {
			data.Balance[k.String()] = v.String()
		}

		if _, exists := data.Balance[types.GetBaseCoin().String()]; !exists {
			data.Balance[types.GetBaseCoin().String()] = "0"
		}

		response.Addresses = append(response.Addresses, data)
	}

	return response, nil
}
