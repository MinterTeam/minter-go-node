package types

import (
	"math/big"
)

type AppState struct {
	Validators  []Validator  `json:"validators,omitempty"`
	Candidates  []Candidate  `json:"candidates,omitempty"`
	Accounts    []Account    `json:"accounts,omitempty"`
	Coins       []Coin       `json:"coins,omitempty"`
	FrozenFunds []FrozenFund `json:"frozen_funds,omitempty"`
	UsedChecks  []UsedCheck  `json:"used_checks,omitempty"`
	MaxGas      uint64       `json:"max_gas"`
}

type Validator struct {
	RewardAddress Address   `json:"reward_address"`
	TotalBipStake *big.Int  `json:"total_bip_stake"`
	PubKey        Pubkey    `json:"pub_key"`
	Commission    uint      `json:"commission"`
	AccumReward   *big.Int  `json:"accum_reward"`
	AbsentTimes   *BitArray `json:"absent_times"`
}

type Candidate struct {
	RewardAddress  Address  `json:"reward_address"`
	OwnerAddress   Address  `json:"owner_address"`
	TotalBipStake  *big.Int `json:"total_bip_stake"`
	PubKey         Pubkey   `json:"pub_key"`
	Commission     uint     `json:"commission"`
	Stakes         []Stake  `json:"stakes"`
	CreatedAtBlock uint     `json:"created_at_block"`
	Status         byte     `json:"status"`
}

type Stake struct {
	Owner    Address    `json:"owner"`
	Coin     CoinSymbol `json:"coin"`
	Value    *big.Int   `json:"value"`
	BipValue *big.Int   `json:"bip_value"`
}

type Coin struct {
	Name           string     `json:"name"`
	Symbol         CoinSymbol `json:"symbol"`
	Volume         *big.Int   `json:"volume"`
	Crr            uint       `json:"crr"`
	ReserveBalance *big.Int   `json:"reserve_balance"`
}

type FrozenFund struct {
	Height       uint64     `json:"height"`
	Address      Address    `json:"address"`
	CandidateKey Pubkey     `json:"candidate_key"`
	Coin         CoinSymbol `json:"coin"`
	Value        *big.Int   `json:"value"`
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
	Value *big.Int   `json:"value"`
}

type Multisig struct {
	Weights   []uint    `json:"weights"`
	Threshold uint      `json:"threshold"`
	Addresses []Address `json:"addresses"`
}
