package accounts

import (
	"bytes"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
	"sort"
)

// Deprecated
func (a *Accounts) ExportV1(state *types.AppState) map[types.CoinID]*big.Int {
	totalSubCoinValue := map[types.CoinID]*big.Int{}
	a.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		addressPath := key[1:]
		if len(addressPath) > types.AddressLength {
			return false
		}

		address := types.BytesToAddress(addressPath)
		account := a.get(address)

		smallValue := true
		subCoinValue := map[types.CoinID]*big.Int{}
		var balance []types.Balance
		for _, b := range a.GetBalances(account.address) {
			if b.Value.Sign() != 1 {
				continue
			}

			if !(smallValue && b.Value.Cmp(big.NewInt(10000000)) == -1) {
				sub, has := subCoinValue[b.Coin.ID]
				if !has {
					sub = big.NewInt(0)
					subCoinValue[b.Coin.ID] = sub
				}
				sub.Add(sub, b.Value)
				smallValue = false
			}

			balance = append(balance, types.Balance{
				Coin:  uint64(b.Coin.ID),
				Value: b.Value.String(),
			})
		}

		// sort balances by coin symbol
		sort.SliceStable(balance, func(i, j int) bool {
			return bytes.Compare(types.CoinID(balance[i].Coin).Bytes(), types.CoinID(balance[j].Coin).Bytes()) == 1
		})

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

		if smallValue && acc.Nonce == 0 && acc.MultisigData == nil {
			for id, sub := range subCoinValue {
				totalSub, has := totalSubCoinValue[id]
				if !has {
					totalSub = big.NewInt(0)
					totalSubCoinValue[id] = totalSub
				}
				totalSub.Add(totalSub, sub)
			}
			return false
		}

		state.Accounts = append(state.Accounts, acc)

		return false
	})

	return totalSubCoinValue
}
