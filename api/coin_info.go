package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/pkg/errors"
)

type CoinInfoResponse struct {
	Name           string           `json:"name"`
	Symbol         types.CoinSymbol `json:"symbol"`
	Volume         string           `json:"volume"`
	Crr            uint             `json:"crr"`
	ReserveBalance string           `json:"reserve_balance"`
}

func CoinInfo(coinSymbol string, height int) (*CoinInfoResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	coin := cState.GetStateCoin(types.StrToCoinSymbol(coinSymbol))
	if coin == nil {
		return nil, errors.New("Coin not found")
	}

	return &CoinInfoResponse{
		Name:           coin.Data().Name,
		Symbol:         coin.Data().Symbol,
		Volume:         coin.Data().Volume.String(),
		Crr:            coin.Data().Crr,
		ReserveBalance: coin.Data().ReserveBalance.String(),
	}, nil
}
