package service

import (
	"context"
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/coreV2/state/waitlist"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

// WaitLists returns the list stakes in waitlist.
func (s *Service) WaitLists(ctx context.Context, req *pb.WaitListsRequest) (*pb.WaitListsResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 {
		cState.Candidates().LoadCandidates()
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	response := new(pb.WaitListsResponse)

	model := cState.WaitList().GetAll()

	if len(model) == 0 {
		return response, nil
	}

	response.Addresses = make([]*pb.WaitListsResponse_Address, 0, len(model))
	for _, items := range model {
		addr := &pb.WaitListsResponse_Address{
			Address: items.Address().String(),
			List:    make([]*pb.WaitListsResponse_Address_Wait, 0, len(items.List)),
		}

		for _, item := range items.List {
			if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
				return nil, timeoutStatus.Err()
			}

			addr.List = append(addr.List, &pb.WaitListsResponse_Address_Wait{
				PublicKey: cState.Candidates().PubKey(item.CandidateId).String(),
				Coin: &pb.Coin{
					Id:     uint64(item.Coin),
					Symbol: cState.Coins().GetCoin(item.Coin).GetFullSymbol(),
				},
				Value: item.Value.String(),
			})
		}
		response.Addresses = append(response.Addresses, addr)
	}

	return response, nil
}

// WaitList returns the list of address stakes in waitlist.
func (s *Service) WaitList(ctx context.Context, req *pb.WaitListRequest) (*pb.WaitListResponse, error) {
	if !strings.HasPrefix(strings.Title(req.Address), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	decodeString, err := hex.DecodeString(req.Address[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	address := types.BytesToAddress(decodeString)

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 {
		cState.Candidates().LoadCandidates()
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	response := new(pb.WaitListResponse)
	var items []*waitlist.Item
	publicKey := req.PublicKey
	if publicKey != "" {
		if !strings.HasPrefix(publicKey, "Mp") {
			return nil, status.Error(codes.InvalidArgument, "public key don't have preffix 'Mp'")
		}
		items = cState.WaitList().GetByAddressAndPubKey(address, types.HexToPubkey(publicKey))
	} else {
		model := cState.WaitList().GetByAddress(address)
		if model == nil {
			return response, nil
		}
		items = model.List
	}
	response.List = make([]*pb.WaitListResponse_Wait, 0, len(items))
	for _, item := range items {
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		response.List = append(response.List, &pb.WaitListResponse_Wait{
			PublicKey: cState.Candidates().PubKey(item.CandidateId).String(),
			Coin: &pb.Coin{
				Id:     uint64(item.Coin),
				Symbol: cState.Coins().GetCoin(item.Coin).GetFullSymbol(),
			},
			Value: item.Value.String(),
		})
	}

	return response, nil
}
