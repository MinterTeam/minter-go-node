package api

import (
	compact_db "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type EventsResponseValidator struct {
	Pubkey        string `json:"pub_key"`
	DaoRole       string `json:"dao"`
	DeveloperRole string `json:"developers"`
	ValidatorRole string `json:"validator"`
	DelegatorRole string `json:"delegator"`
}

func GroupedEvents(height int64) ([]*EventsResponseValidator, error) {

	recipients := map[string]struct{}{"Validator": {}, "Delegator": {}, "DAO": {}, "Developers": {}}

	if height == 0 {
		height = int64(blockchain.Height())
	}

	tmVals, err := client.Validators(&height, 1, 100)
	if err != nil {
		return nil, err
	}

	mapPubkeyEvents := map[types.Pubkey][]*compact_db.RewardEvent{}
	events := blockchain.GetEventsDB().LoadEvents(uint32(height))
	for _, event := range events {
		e, ok := event.(*compact_db.RewardEvent)
		if !ok {
			continue
		}
		if _, ok := recipients[e.Role]; !ok {
			continue
		}
		mapPubkeyEvents[e.ValidatorPubKey] = append(mapPubkeyEvents[e.ValidatorPubKey], e)
	}

	var responseValidators []*EventsResponseValidator
	for _, val := range tmVals.Validators {
		pubKey := types.BytesToPubkey(val.PubKey.Bytes()[5:])
		events, ok := mapPubkeyEvents[pubKey]
		if !ok {
			continue
		}

		validator := &EventsResponseValidator{
			Pubkey: pubKey.String(),
		}

		delegatorRoleValue := big.NewInt(0)
		for _, event := range events {
			eventDelegatorAmount := big.NewInt(0)
			switch event.Role {
			case "Validator":
				validator.ValidatorRole = event.Role
			case "Delegator":
				eventDelegatorAmount.SetString(event.Amount, 10)
			case "DAO":
				validator.DaoRole = event.Role
			case "Developers":
				validator.DeveloperRole = event.Role
			default:
				continue
			}
			delegatorRoleValue.Add(delegatorRoleValue, eventDelegatorAmount)
		}
		if validator.DaoRole == "" {
			continue
		}

		validator.DelegatorRole = delegatorRoleValue.String()

		responseValidators = append(responseValidators, validator)
	}

	return responseValidators, nil
}
