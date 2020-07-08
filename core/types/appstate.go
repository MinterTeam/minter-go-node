package types

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
)

type AppState struct {
	Note         string       `json:"note"`
	StartHeight  uint64       `json:"start_height"`
	Validators   []Validator  `json:"validators,omitempty"`
	Candidates   []Candidate  `json:"candidates,omitempty"`
	Accounts     []Account    `json:"accounts,omitempty"`
	Coins        []Coin       `json:"coins,omitempty"`
	FrozenFunds  []FrozenFund `json:"frozen_funds,omitempty"`
	HaltBlocks   []HaltBlock  `json:"halt_blocks,omitempty"`
	UsedChecks   []UsedCheck  `json:"used_checks,omitempty"`
	MaxGas       uint64       `json:"max_gas"`
	TotalSlashed string       `json:"total_slashed"`
}

func (s *AppState) Verify() error {
	if !helpers.IsValidBigInt(s.TotalSlashed) {
		return fmt.Errorf("total slashed is not valid BigInt")
	}

	if len(s.Validators) < 1 {
		return fmt.Errorf("there should be at least one validator")
	}

	validators := map[Pubkey]struct{}{}
	for _, val := range s.Validators {
		// check for validators duplication
		if _, exists := validators[val.PubKey]; exists {
			return fmt.Errorf("duplicated validator %s", val.PubKey.String())
		}

		validators[val.PubKey] = struct{}{}

		// search for candidate
		foundCandidate := false
		for _, candidate := range s.Candidates {
			if candidate.PubKey == val.PubKey {
				foundCandidate = true
				break
			}
		}

		if !foundCandidate {
			return fmt.Errorf("candidate for validator %s not found", val.PubKey.String())
		}

		// basic checks
		if !helpers.IsValidBigInt(val.TotalBipStake) {
			return fmt.Errorf("total bip stake of validator %s is not valid", val.PubKey.String())
		}

		if !helpers.IsValidBigInt(val.AccumReward) {
			return fmt.Errorf("accum reward of validator %s is not valid", val.PubKey.String())
		}

		if val.AbsentTimes == nil {
			return fmt.Errorf("absent times of validator %s is not valid", val.PubKey.String())
		}
	}

	accounts := map[Address]struct{}{}
	for _, acc := range s.Accounts {
		// check for account duplication
		if _, exists := accounts[acc.Address]; exists {
			return fmt.Errorf("duplicated account %s", acc.Address.String())
		}

		accounts[acc.Address] = struct{}{}

		for _, bal := range acc.Balance {
			if !helpers.IsValidBigInt(bal.Value) {
				return fmt.Errorf("not valid balance for account %s", acc.Address.String())
			}

			if !bal.Coin.IsBaseCoin() {
				// check not existing coins
				foundCoin := false
				for _, coin := range s.Coins {
					if coin.Symbol == bal.Coin {
						foundCoin = true
						break
					}
				}

				if !foundCoin {
					return fmt.Errorf("coin %s not found", bal.Coin)
				}
			}
		}
	}

	for _, candidate := range s.Candidates {
		stakes := map[string]struct{}{}
		for _, stake := range candidate.Stakes {
			// check duplicated stakes
			key := fmt.Sprintf("%s:%s", stake.Owner.String(), stake.Coin.String())
			if _, exists := stakes[key]; exists {
				return fmt.Errorf("duplicated stake %s", key)
			}
			stakes[key] = struct{}{}

			// check not existing coins
			if !stake.Coin.IsBaseCoin() {
				foundCoin := false
				for _, coin := range s.Coins {
					if coin.Symbol == stake.Coin {
						foundCoin = true
						break
					}
				}

				if !foundCoin {
					return fmt.Errorf("coin %s not found", stake.Coin)
				}
			}
		}
	}

	coins := map[CoinSymbol]struct{}{}
	for _, coin := range s.Coins {
		if coin.Symbol.IsBaseCoin() {
			return fmt.Errorf("base coin should not be declared")
		}

		// check duplicated coins
		if _, exists := coins[coin.Symbol]; exists {
			return fmt.Errorf("duplicated coin %s", coin.Symbol)
		}
		coins[coin.Symbol] = struct{}{}

		// check coins' volume
		volume := big.NewInt(0)
		for _, ff := range s.FrozenFunds {
			if ff.Coin == coin.Symbol {
				volume.Add(volume, helpers.StringToBigInt(ff.Value))
			}
		}

		for _, candidate := range s.Candidates {
			for _, stake := range candidate.Stakes {
				if stake.Coin == coin.Symbol {
					volume.Add(volume, helpers.StringToBigInt(stake.Value))
				}
			}
		}

		for _, account := range s.Accounts {
			for _, bal := range account.Balance {
				if bal.Coin == coin.Symbol {
					volume.Add(volume, helpers.StringToBigInt(bal.Value))
				}
			}
		}

		if volume.Cmp(helpers.StringToBigInt(coin.Volume)) != 0 {
			return fmt.Errorf("wrong coin %s volume", coin.Symbol.String())
		}
	}

	for _, ff := range s.FrozenFunds {
		if !helpers.IsValidBigInt(ff.Value) {
			return fmt.Errorf("wrong frozen fund value: %s", ff.Value)
		}

		// check not existing coins
		if !ff.Coin.IsBaseCoin() {
			foundCoin := false
			for _, coin := range s.Coins {
				if coin.Symbol == ff.Coin {
					foundCoin = true
					break
				}
			}

			if !foundCoin {
				return fmt.Errorf("coin %s not found", ff.Coin)
			}
		}
	}

	// check used checks length
	for _, check := range s.UsedChecks {
		b, err := hex.DecodeString(string(check))
		if err != nil {
			return err
		}

		if len(b) != 32 {
			return fmt.Errorf("wrong used check size %s", check)
		}
	}

	return nil
}

type Validator struct {
	TotalBipStake string    `json:"total_bip_stake"`
	PubKey        Pubkey    `json:"pub_key"`
	AccumReward   string    `json:"accum_reward"`
	AbsentTimes   *BitArray `json:"absent_times"`
}

type Candidate struct {
	RewardAddress Address `json:"reward_address"`
	OwnerAddress  Address `json:"owner_address"`
	TotalBipStake string  `json:"total_bip_stake"`
	PubKey        Pubkey  `json:"pub_key"`
	Commission    uint    `json:"commission"`
	Stakes        []Stake `json:"stakes"`
	Updates       []Stake `json:"updates"`
	Status        byte    `json:"status"`
}

type Stake struct {
	Owner    Address    `json:"owner"`
	Coin     CoinSymbol `json:"coin"`
	Value    string     `json:"value"`
	BipValue string     `json:"bip_value"`
}

type Coin struct {
	Name      string     `json:"name"`
	Symbol    CoinSymbol `json:"symbol"`
	Volume    string     `json:"volume"`
	Crr       uint       `json:"crr"`
	Reserve   string     `json:"reserve"`
	MaxSupply string     `json:"max_supply"`
}

type FrozenFund struct {
	Height       uint64     `json:"height"`
	Address      Address    `json:"address"`
	CandidateKey *Pubkey    `json:"candidate_key,omitempty"`
	Coin         CoinSymbol `json:"coin"`
	Value        string     `json:"value"`
}

type UsedCheck string

type Account struct {
	Address      Address   `json:"address"`
	Balance      []Balance `json:"balance"`
	Nonce        uint64    `json:"nonce"`
	MultisigData *Multisig `json:"multisig_data,omitempty"`
}

type Balance struct {
	Coin  CoinSymbol `json:"coin"`
	Value string     `json:"value"`
}

type Multisig struct {
	Weights   []uint    `json:"weights"`
	Threshold uint      `json:"threshold"`
	Addresses []Address `json:"addresses"`
}

type HaltBlock struct {
	Height       uint64 `json:"height"`
	CandidateKey Pubkey `json:"candidate_key"`
}
