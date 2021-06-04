package types

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/MinterTeam/minter-go-node/helpers"
)

type AppState struct {
	Note                string           `json:"note"`
	Validators          []Validator      `json:"validators,omitempty"`
	Candidates          []Candidate      `json:"candidates,omitempty"`
	BlockListCandidates []Pubkey         `json:"block_list_candidates,omitempty"`
	Waitlist            []Waitlist       `json:"waitlist,omitempty"`
	Pools               []Pool           `json:"pools,omitempty"`
	NextOrderID         uint64           `json:"next_order_id"`
	Accounts            []Account        `json:"accounts,omitempty"`
	Coins               []Coin           `json:"coins,omitempty"`
	FrozenFunds         []FrozenFund     `json:"frozen_funds,omitempty"`
	HaltBlocks          []HaltBlock      `json:"halt_blocks,omitempty"`
	Commission          Commission       `json:"commission,omitempty"`
	CommissionVotes     []CommissionVote `json:"commission_votes,omitempty"`
	UpdateVotes         []UpdateVote     `json:"update_votes,omitempty"`
	UsedChecks          []UsedCheck      `json:"used_checks,omitempty"`
	MaxGas              uint64           `json:"max_gas"`
	TotalSlashed        string           `json:"total_slashed"`
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

			coinID := CoinID(bal.Coin)
			if !coinID.IsBaseCoin() {
				// check not existing coins
				foundCoin := false
				for _, coin := range s.Coins {
					id := CoinID(coin.ID)
					if id == coinID {
						foundCoin = true
						break
					}
				}

				if !foundCoin {
					return fmt.Errorf("coin %s not found", coinID)
				}
			}
		}
	}

	for _, candidate := range s.Candidates {
		stakes := map[string]struct{}{}
		for _, stake := range candidate.Stakes {
			// check duplicated stakes
			coinID := CoinID(stake.Coin)
			key := fmt.Sprintf("%s:%s", stake.Owner.String(), coinID.String())
			if _, exists := stakes[key]; exists {
				return fmt.Errorf("duplicated stake %s", key)
			}
			stakes[key] = struct{}{}

			// check not existing coins
			if !coinID.IsBaseCoin() {
				foundCoin := false
				for _, coin := range s.Coins {
					id := CoinID(coin.ID)
					if id == coinID {
						foundCoin = true
						break
					}
				}

				if !foundCoin {
					return fmt.Errorf("coin %s not found", coinID)
				}
			}
		}
	}

	coins := map[uint64]struct{}{}
	for _, coin := range s.Coins {
		if coin.Symbol.IsBaseCoin() {
			return fmt.Errorf("base coin should not be declared")
		}

		// check duplicated coins
		if _, exists := coins[coin.ID]; exists {
			return fmt.Errorf("duplicated coin %s", coin.Symbol)
		}

		coins[coin.ID] = struct{}{}

		// check coins' volume
		volume := big.NewInt(0)

		for _, account := range s.Accounts {
			for _, bal := range account.Balance {
				if bal.Coin == coin.ID {
					volume.Add(volume, helpers.StringToBigInt(bal.Value))
				}
			}
		}

		for _, swap := range s.Pools {
			if coin.ID != swap.Coin0 && coin.ID != swap.Coin1 {
				continue
			}
			if swap.Coin0 == coin.ID {
				volume.Add(volume, helpers.StringToBigInt(swap.Reserve0))

			}
			if swap.Coin1 == coin.ID {
				volume.Add(volume, helpers.StringToBigInt(swap.Reserve1))

			}
			for _, order := range swap.Orders {
				if !order.IsSale {
					if swap.Coin0 == coin.ID {
						volume.Add(volume, helpers.StringToBigInt(order.Volume0))
					}
				} else {
					if swap.Coin1 == coin.ID {
						volume.Add(volume, helpers.StringToBigInt(order.Volume1))
					}
				}
			}

		}

		if coin.Crr == 0 {
			if volume.Cmp(helpers.StringToBigInt(coin.Volume)) != 0 {
				return fmt.Errorf("wrong token %s (%d) volume (%s)", coin.Symbol.String(), coin.ID, big.NewInt(0).Sub(volume, helpers.StringToBigInt(coin.Volume)))
			}
			continue
		}

		for _, ff := range s.FrozenFunds {
			if ff.Coin == coin.ID {
				volume.Add(volume, helpers.StringToBigInt(ff.Value))
			}
		}

		for _, candidate := range s.Candidates {
			for _, stake := range candidate.Stakes {
				if stake.Coin == coin.ID {
					volume.Add(volume, helpers.StringToBigInt(stake.Value))
				}
			}

			for _, stake := range candidate.Updates {
				if stake.Coin == coin.ID {
					volume.Add(volume, helpers.StringToBigInt(stake.Value))
				}
			}
		}

		for _, wl := range s.Waitlist {
			if wl.Coin == coin.ID {
				volume.Add(volume, helpers.StringToBigInt(wl.Value))
			}
		}

		if volume.Cmp(helpers.StringToBigInt(coin.Volume)) != 0 {
			return fmt.Errorf("wrong coin %s volume (%s)", coin.Symbol.String(), big.NewInt(0).Sub(volume, helpers.StringToBigInt(coin.Volume)))
		}
	}

	for _, wl := range s.Waitlist {
		if !helpers.IsValidBigInt(wl.Value) {
			return fmt.Errorf("wrong waitlist value: %s", wl.Value)
		}

		// check not existing coins
		coinID := CoinID(wl.Coin)
		if !coinID.IsBaseCoin() {
			foundCoin := false
			for _, coin := range s.Coins {
				if CoinID(coin.ID) == coinID {
					foundCoin = true
					break
				}
			}

			if !foundCoin {
				return fmt.Errorf("coin %s not found", coinID)
			}
		}
	}

	for _, ff := range s.FrozenFunds {
		if !helpers.IsValidBigInt(ff.Value) {
			return fmt.Errorf("wrong frozen fund value: %s", ff.Value)
		}

		// check not existing coins
		coinID := CoinID(ff.Coin)
		if !coinID.IsBaseCoin() {
			foundCoin := false
			for _, coin := range s.Coins {
				id := CoinID(coin.ID)
				if id == coinID {
					foundCoin = true
					break
				}
			}

			if !foundCoin {
				return fmt.Errorf("coin %s not found", coinID)
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
	PubKey        Pubkey    `json:"public_key"`
	AccumReward   string    `json:"accum_reward"`
	AbsentTimes   *BitArray `json:"absent_times"`
}

type Candidate struct {
	ID                       uint64  `json:"id"`
	RewardAddress            Address `json:"reward_address"`
	OwnerAddress             Address `json:"owner_address"`
	ControlAddress           Address `json:"control_address"`
	TotalBipStake            string  `json:"total_bip_stake"`
	PubKey                   Pubkey  `json:"public_key"`
	Commission               uint64  `json:"commission"`
	Stakes                   []Stake `json:"stakes,omitempty"`
	Updates                  []Stake `json:"updates,omitempty"`
	Status                   uint64  `json:"status"`
	JailedUntil              uint64  `json:"jailed_until,omitempty"`
	LastEditCommissionHeight uint64  `json:"last_edit_commission_height,omitempty"`
}

type Stake struct {
	Owner    Address `json:"owner"`
	Coin     uint64  `json:"coin"`
	Value    string  `json:"value"`
	BipValue string  `json:"bip_value"`
}

type Waitlist struct {
	CandidateID uint64  `json:"candidate_id"`
	Owner       Address `json:"owner"`
	Coin        uint64  `json:"coin"`
	Value       string  `json:"value"`
}
type Order struct {
	IsSale  bool
	Volume0 string  `json:"volume0"`
	Volume1 string  `json:"volume1"`
	ID      uint64  `json:"id"`
	Owner   Address `json:"owner"`
}
type Pool struct {
	Coin0    uint64  `json:"coin0"`
	Coin1    uint64  `json:"coin1"`
	Reserve0 string  `json:"reserve0"`
	Reserve1 string  `json:"reserve1"`
	ID       uint64  `json:"id"`
	Orders   []Order `json:"orders"`
}

type Coin struct {
	ID           uint64     `json:"id"`
	Name         string     `json:"name"`
	Symbol       CoinSymbol `json:"symbol"`
	Volume       string     `json:"volume"`
	Crr          uint64     `json:"crr,omitempty"`
	Reserve      string     `json:"reserve,omitempty"`
	MaxSupply    string     `json:"max_supply"`
	Version      uint64     `json:"version,omitempty"`
	OwnerAddress *Address   `json:"owner_address,omitempty"`
	Mintable     bool       `json:"mintable,omitempty"`
	Burnable     bool       `json:"burnable,omitempty"`
}

type FrozenFund struct {
	Height       uint64  `json:"height"`
	Address      Address `json:"address"`
	CandidateKey *Pubkey `json:"candidate_key,omitempty"`
	CandidateID  uint64  `json:"candidate_id,omitempty"`
	Coin         uint64  `json:"coin"`
	Value        string  `json:"value"`
	// MoveToCandidateID *uint64 `json:"move_to_candidate_id,omitempty"`
}

type UsedCheck string

type Account struct {
	Address      Address   `json:"address"`
	Balance      []Balance `json:"balance,omitempty"`
	Nonce        uint64    `json:"nonce"`
	MultisigData *Multisig `json:"multisig_data,omitempty"`
}

type Balance struct {
	Coin  uint64 `json:"coin"`
	Value string `json:"value"`
}

type Multisig struct {
	Weights   []uint64  `json:"weights"`
	Threshold uint64    `json:"threshold"`
	Addresses []Address `json:"addresses"`
}

type HaltBlock struct {
	Height       uint64 `json:"height"`
	CandidateKey Pubkey `json:"candidate_key"`
}
type CommissionVote struct {
	Height     uint64     `json:"height"`
	Votes      []Pubkey   `json:"votes"`
	Commission Commission `json:"commission"`
}

type UpdateVote struct {
	Height  uint64   `json:"height"`
	Votes   []Pubkey `json:"votes"`
	Version string   `json:"version"`
}

type Commission struct {
	Coin                    uint64 `json:"coin"`
	PayloadByte             string `json:"payload_byte"`
	Send                    string `json:"send"`
	BuyBancor               string `json:"buy_bancor"`
	SellBancor              string `json:"sell_bancor"`
	SellAllBancor           string `json:"sell_all_bancor"`
	BuyPoolBase             string `json:"buy_pool_base"`
	BuyPoolDelta            string `json:"buy_pool_delta"`
	SellPoolBase            string `json:"sell_pool_base"`
	SellPoolDelta           string `json:"sell_pool_delta"`
	SellAllPoolBase         string `json:"sell_all_pool_base"`
	SellAllPoolDelta        string `json:"sell_all_pool_delta"`
	CreateTicker3           string `json:"create_ticker3"`
	CreateTicker4           string `json:"create_ticker4"`
	CreateTicker5           string `json:"create_ticker5"`
	CreateTicker6           string `json:"create_ticker6"`
	CreateTicker7_10        string `json:"create_ticker7_10"`
	CreateCoin              string `json:"create_coin"`
	CreateToken             string `json:"create_token"`
	RecreateCoin            string `json:"recreate_coin"`
	RecreateToken           string `json:"recreate_token"`
	DeclareCandidacy        string `json:"declare_candidacy"`
	Delegate                string `json:"delegate"`
	Unbond                  string `json:"unbond"`
	RedeemCheck             string `json:"redeem_check"`
	SetCandidateOn          string `json:"set_candidate_on"`
	SetCandidateOff         string `json:"set_candidate_off"`
	CreateMultisig          string `json:"create_multisig"`
	MultisendBase           string `json:"multisend_base"`
	MultisendDelta          string `json:"multisend_delta"`
	EditCandidate           string `json:"edit_candidate"`
	SetHaltBlock            string `json:"set_halt_block"`
	EditTickerOwner         string `json:"edit_ticker_owner"`
	EditMultisig            string `json:"edit_multisig"`
	EditCandidatePublicKey  string `json:"edit_candidate_public_key"`
	CreateSwapPool          string `json:"create_swap_pool"`
	AddLiquidity            string `json:"add_liquidity"`
	RemoveLiquidity         string `json:"remove_liquidity"`
	EditCandidateCommission string `json:"edit_candidate_commission"`
	MintToken               string `json:"mint_token"`
	BurnToken               string `json:"burn_token"`
	VoteCommission          string `json:"vote_commission"`
	VoteUpdate              string `json:"vote_update"`
}
