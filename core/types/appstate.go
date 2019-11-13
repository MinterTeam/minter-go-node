package types

type AppState struct {
	Note         string       `json:"note"`
	StartHeight  uint64       `json:"start_height"`
	Validators   []Validator  `json:"validators,omitempty"`
	Candidates   []Candidate  `json:"candidates,omitempty"`
	Accounts     []Account    `json:"accounts,omitempty"`
	Coins        []Coin       `json:"coins,omitempty"`
	FrozenFunds  []FrozenFund `json:"frozen_funds,omitempty"`
	UsedChecks   []UsedCheck  `json:"used_checks,omitempty"`
	MaxGas       uint64       `json:"max_gas"`
	TotalSlashed string       `json:"total_slashed"`
}

type Validator struct {
	RewardAddress Address   `json:"reward_address"`
	TotalBipStake string    `json:"total_bip_stake"`
	PubKey        Pubkey    `json:"pub_key"`
	Commission    uint      `json:"commission"`
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
