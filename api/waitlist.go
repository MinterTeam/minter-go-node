package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type WaitlistResponse struct {
	List []*Wait `json:"list"`
}

type Wait struct {
	Coin  Coin   `json:"coin"`
	Value string `json:"value"`
}

func Waitlist(pubkey types.Pubkey, address types.Address, height int) (*WaitlistResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	response := new(WaitlistResponse)
	items := cState.WaitList().GetByAddressAndPubKey(address, pubkey)
	response.List = make([]*Wait, 0, len(items))
	for _, item := range items {
		response.List = append(response.List, &Wait{
			Coin: Coin{
				ID:     item.Coin.Uint32(),
				Symbol: cState.Coins().GetCoin(item.Coin).GetFullSymbol(),
			},
			Value: item.Value.String(),
		})
	}

	return response, nil
}
