package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	rpctypes "github.com/MinterTeam/minter-go-node/rpc/lib/types"
)

func CoinIdInfo(id int, height int) (*CoinInfoResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

	coin := cState.Coins().GetCoin(types.CoinID(id))
	if coin == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin not found"}
	}

	var ownerAddress *types.Address
	info := cState.Coins().GetSymbolInfo(coin.Symbol())
	if info != nil && info.OwnerAddress() != nil {
		ownerAddress = info.OwnerAddress()
	}

	return &CoinInfoResponse{
		ID:             coin.ID().Uint32(),
		Name:           coin.Name(),
		Symbol:         coin.GetFullSymbol(),
		Volume:         coin.Volume().String(),
		Crr:            coin.Crr(),
		ReserveBalance: coin.Reserve().String(),
		MaxSupply:      coin.MaxSupply().String(),
		OwnerAddress:   ownerAddress,
	}, nil
}
