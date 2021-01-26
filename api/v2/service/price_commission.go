package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/core/state/coins"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PriceCommission returns current tx commissions
func (s *Service) PriceCommission(ctx context.Context, req *pb.PriceCommissionRequest) (*pb.PriceCommissionResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	price := cState.Commission().GetCommissions()

	return priceCommissionResponse(price, cState.Coins().GetCoin(price.Coin)), nil
}

func priceCommissionResponse(price *commission.Price, coin *coins.Model) *pb.PriceCommissionResponse {
	return &pb.PriceCommissionResponse{
		Coin: &pb.Coin{
			Id:     uint64(price.Coin),
			Symbol: coin.GetFullSymbol(),
		},
		PayloadByte:             price.PayloadByte.String(),
		Send:                    price.Send.String(),
		BuyBancor:               price.BuyBancor.String(),
		SellBancor:              price.SellBancor.String(),
		SellAllBancor:           price.SellAllBancor.String(),
		BuyPool:                 price.BuyPool.String(),
		SellPool:                price.SellPool.String(),
		SellAllPool:             price.SellAllPool.String(),
		CreateTicker3:           price.CreateTicker3.String(),
		CreateTicker4:           price.CreateTicker4.String(),
		CreateTicker5:           price.CreateTicker5.String(),
		CreateTicker6:           price.CreateTicker6.String(),
		CreateTicker7_10:        price.CreateTicker7to10.String(),
		CreateCoin:              price.CreateCoin.String(),
		CreateToken:             price.CreateToken.String(),
		RecreateCoin:            price.RecreateCoin.String(),
		RecreateToken:           price.RecreateToken.String(),
		DeclareCandidacy:        price.DeclareCandidacy.String(),
		Delegate:                price.Delegate.String(),
		Unbond:                  price.Unbond.String(),
		RedeemCheck:             price.RedeemCheck.String(),
		SetCandidateOn:          price.SetCandidateOn.String(),
		SetCandidateOff:         price.SetCandidateOff.String(),
		CreateMultisig:          price.CreateMultisig.String(),
		MultisendDelta:          price.MultisendDelta.String(),
		EditCandidate:           price.EditCandidate.String(),
		SetHaltBlock:            price.SetHaltBlock.String(),
		EditTickerOwner:         price.EditTickerOwner.String(),
		EditMultisig:            price.EditMultisig.String(),
		PriceVote:               price.PriceVote.String(),
		EditCandidatePublicKey:  price.EditCandidatePublicKey.String(),
		AddLiquidity:            price.AddLiquidity.String(),
		RemoveLiquidity:         price.RemoveLiquidity.String(),
		EditCandidateCommission: price.EditCandidateCommission.String(),
		MoveStake:               price.MoveStake.String(),
		EditTokenEmission:       price.EditTokenEmission.String(),
		PriceCommission:         price.PriceCommission.String(),
		UpdateNetwork:           price.UpdateNetwork.String(),
	}
}
