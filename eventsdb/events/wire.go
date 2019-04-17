package events

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

type Role byte

func (r Role) String() string {
	switch r {
	case RoleValidator:
		return "Validator"
	case RoleDelegator:
		return "Delegator"
	case RoleDAO:
		return "DAO"
	case RoleDevelopers:
		return "Developers"
	}

	return "Undefined"
}

const (
	RoleValidator Role = iota
	RoleDelegator
	RoleDAO
	RoleDevelopers
)

type Event interface{}
type Events []Event
