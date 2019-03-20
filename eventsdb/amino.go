package eventsdb

import "github.com/MinterTeam/go-amino"

func RegisterAminoEvents(codec *amino.Codec) {
	codec.RegisterInterface((*Event)(nil), nil)
	codec.RegisterConcrete(RewardEvent{},
		"minter/RewardEvent", nil)
	codec.RegisterConcrete(SlashEvent{},
		"minter/SlashEvent", nil)
	codec.RegisterConcrete(UnbondEvent{},
		"minter/UnbondEvent", nil)
	codec.RegisterConcrete(CoinLiquidationEvent{},
		"minter/CoinLiquidationEvent", nil)
}
