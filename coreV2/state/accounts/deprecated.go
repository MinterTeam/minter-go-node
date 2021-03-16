package accounts

import (
	"github.com/MinterTeam/minter-go-node/coreV2/dao"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type MaxCoinVolume struct {
	Owner  types.Address
	Volume *big.Int
}

// Deprecated
func (a *Accounts) ExportV1(state *types.AppState, subBipValueFromDAO *big.Int) (map[types.CoinID]*big.Int, map[types.CoinID]*MaxCoinVolume) {
	totalSubCoinValue := map[types.CoinID]*big.Int{}
	maxVolume := map[types.CoinID]*MaxCoinVolume{}
	a.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		addressPath := key[1:]
		if len(addressPath) > types.AddressLength {
			return false
		}

		address := types.BytesToAddress(addressPath)
		account := a.get(address)

		subCoinValue := map[types.CoinID]*big.Int{}
		var balance []types.Balance
		for _, b := range a.GetBalancesV1(account.address) {
			if b.Value.Cmp(big.NewInt(10000000)) == -1 {
				if b.Value.Sign() == 0 {
					continue
				}
				if account.Nonce == 0 && !account.IsMultisig() {
					sub, has := subCoinValue[b.Coin.ID]
					if !has {
						sub = big.NewInt(0)
						subCoinValue[b.Coin.ID] = sub
					}
					sub.Add(sub, b.Value)
					continue
				}
			}

			value := b.Value
			if address == dao.Address && b.Coin.ID == types.GetBaseCoinID() {
				value.Sub(value, subBipValueFromDAO)
			}

			balance = append(balance, types.Balance{
				Coin:  uint64(b.Coin.ID),
				Value: value.String(),
			})

			tmp := big.NewInt(0).Set(value)
			if volume, ok := maxVolume[b.Coin.ID]; ok {
				if volume.Volume.Cmp(tmp) != 1 {
					continue
				}
			}

			maxVolume[b.Coin.ID] = &MaxCoinVolume{
				Owner:  account.address,
				Volume: tmp,
			}
		}

		acc := types.Account{
			Address: account.address,
			Balance: balance,
			Nonce:   account.Nonce,
		}

		if account.IsMultisig() {
			var weights []uint64
			for _, weight := range account.MultisigData.Weights {
				weights = append(weights, uint64(weight))
			}
			acc.MultisigData = &types.Multisig{
				Weights:   weights,
				Threshold: uint64(account.MultisigData.Threshold),
				Addresses: account.MultisigData.Addresses,
			}
		}

		if acc.Nonce == 0 && acc.MultisigData == nil {
			for id, sub := range subCoinValue {
				totalSub, has := totalSubCoinValue[id]
				if !has {
					totalSub = big.NewInt(0)
					totalSubCoinValue[id] = totalSub
				}
				totalSub.Add(totalSub, sub)
			}

			if len(acc.Balance) == 0 {
				return false
			}
		}

		state.Accounts = append(state.Accounts, acc)

		return false
	})

	return totalSubCoinValue, maxVolume
}

// Deprecated
func (a *Accounts) GetBalancesV1(address types.Address) []Balance {
	account := a.getOrNew(address)

	account.lock.RLock()
	coins := account.coins
	account.lock.RUnlock()

	balances := make([]Balance, 0, len(coins))
	for _, id := range coins {
		balances = append(balances, Balance{
			Coin:  *a.bus.Coins().GetCoinV1(id),
			Value: a.GetBalance(address, id),
		})
	}

	return balances
}
