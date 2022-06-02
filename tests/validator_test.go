package tests

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
	"testing"
)

func TestEqualValidatorCmpBalances(t *testing.T) {
	var addresses []types.Address
	var users []*User
	for i := 0; i < 10; i++ {
		owner := CreateAddress()
		users = append(users, owner)
		addresses = append(addresses, owner.address)
	}

	helper := NewHelper(DefaultAppState(addresses...))

	var initialBalances = make(map[types.Address]map[types.CoinID]*big.Int)
	for i, address := range addresses {
		for _, addr := range []types.Address{address, getCustomAddress(i), getRewardAddress(i)} {
			initialBalances[addr] = make(map[types.CoinID]*big.Int)
			for _, balance := range helper.app.CurrentState().Accounts().GetBalances(addr) {
				initialBalances[addr][balance.Coin.ID] = balance.Value
			}
		}
	}

	t.Run("rewards", func(t *testing.T) {
		testBlocks(t, helper, 50)

		for i, address := range addresses {
			for _, addr := range []types.Address{address, getCustomAddress(i), getRewardAddress(i)} {
				for coin, balance := range initialBalances[addr] {
					if balance.String() != helper.app.CurrentState().Accounts().GetBalance(address, coin).String() {
						t.Error("account"+address.String()+"balance is diff", balance)
					}
				}
			}
		}
	})

	t.Run("delegate", func(t *testing.T) {
		delegatedCoin := types.CoinID(1)
		{
			var txs []transaction.Transaction
			for i, delegator := range users {
				txs = append(txs, helper.CreateTx(delegator.privateKey, transaction.DelegateDataV260{
					PubKey: getValidatorAddress(i),
					Coin:   delegatedCoin,
					Value:  initialBalances[delegator.address][delegatedCoin],
				}, types.USDTID))
			}

			_, results := helper.NextBlock(txs...)
			for _, resp := range results {
				if resp.Code != code.OK {
					t.Fatalf("Response code is not OK: %d, %s", resp.Code, resp.Log)
				}
			}
		}
		testBlocks(t, helper, 50)

		var equalRewards = make(map[int]*big.Int)
		for i, address := range addresses {
			candidate := helper.app.CurrentState().Candidates().GetCandidate(getValidatorAddress(i))
			if candidate.Status != 2 {
				t.Error("status is", candidate.Status)
			}
			delegateCoin1Balance := helper.app.CurrentState().Candidates().GetStakeValueOfAddress(getValidatorAddress(i), address, delegatedCoin)
			if delegateCoin1Balance.String() != initialBalances[address][delegatedCoin].String() {
				t.Errorf("validator "+getValidatorAddress(i).String()+" address "+address.String()+"delegate Coin1 Volume wanted %s, got %s", initialBalances[address][delegatedCoin].String(), delegateCoin1Balance)
			}
			for j, addr := range []types.Address{address, getCustomAddress(i), getRewardAddress(i)} {
				rewardAddressBalance := helper.app.CurrentState().Candidates().GetStakeValueOfAddress(getValidatorAddress(i), addr, types.GetBaseCoinID())
				if equalRewards[j] == nil {
					equalRewards[j] = rewardAddressBalance
				}
				if rewardAddressBalance.String() != equalRewards[j].String() {
					t.Error("validator "+getValidatorAddress(i).String()+" address "+address.String()+" diff reward", i, rewardAddressBalance.String())
				}
			}
			balance := helper.app.CurrentState().Accounts().GetBalance(address, delegatedCoin).String()
			if balance != "0" {
				t.Error(helper.app.CurrentState().Coins().GetCoin(delegatedCoin).Symbol().String(), address.String(), "balance is", balance)
			}
		}
	})

}
