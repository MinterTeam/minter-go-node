package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"math/big"
)

type CoinInfoResponse struct {
	Name           string           `json:"name"`
	Symbol         types.CoinSymbol `json:"symbol"`
	Volume         *big.Int         `json:"volume"`
	Crr            uint             `json:"crr"`
	ReserveBalance *big.Int         `json:"reserve_balance"`
}

func CoinInfo(coinSymbol string, height int) (*CoinInfoResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	coin := cState.Coins.GetCoin(types.StrToCoinSymbol(coinSymbol))
	if coin == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin not found"}
	}

	return &CoinInfoResponse{
		Name:           coin.Name(),
		Symbol:         coin.Symbol(),
		Volume:         coin.Volume(),
		Crr:            coin.Crr(),
		ReserveBalance: coin.Reserve(),
	}, nil
}
