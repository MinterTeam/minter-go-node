package events

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

// Event type names
const (
	TypeRewardEvent            = "minter/RewardEvent"
	TypeSlashEvent             = "minter/SlashEvent"
	TypeUnbondEvent            = "minter/UnbondEvent"
	TypeStakeKickEvent         = "minter/StakeKickEvent"
	TypeStakeMoveEvent         = "minter/StakeMoveEvent"
	TypeUpdateNetworkEvent     = "minter/UpdateNetworkEvent"
	TypeUpdateCommissionsEvent = "minter/UpdateCommissionsEvent"
)

type Stake interface {
	AddressString() string
	ValidatorPubKeyString() string
	validatorPubKey() types.Pubkey
	address() types.Address
	convert(pubKeyID uint16, addressID uint32) compact
}

type Event interface {
	Type() string
	// event()
}

type stake interface {
	compile(pubKey [32]byte, address [20]byte) Event
	addressID() uint32
	pubKeyID() uint16
}

type compact interface {
	// compact()
}

type Events []Event

type Role byte

const (
	RoleValidator Role = iota
	RoleDelegator
	RoleDAO
	RoleDevelopers
)

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

	panic(fmt.Sprintf("undefined role: %d", r))
}

func NewRole(r string) Role {
	switch r {
	case "Validator":
		return RoleValidator
	case "Delegator":
		return RoleDelegator
	case "DAO":
		return RoleDAO
	case "Developers":
		return RoleDevelopers
	}

	panic("undefined role: " + r)
}

type reward struct {
	Role      Role
	AddressID uint32
	Amount    []byte
	PubKeyID  uint16
}

func (r *reward) compile(pubKey [32]byte, address [20]byte) Event {
	event := new(RewardEvent)
	event.ValidatorPubKey = pubKey
	event.Address = address
	event.Role = r.Role.String()
	event.Amount = big.NewInt(0).SetBytes(r.Amount).String()
	return event
}

func (r *reward) addressID() uint32 {
	return r.AddressID
}

func (r *reward) pubKeyID() uint16 {
	return r.PubKeyID
}

type RewardEvent struct {
	Role            string        `json:"role"`
	Address         types.Address `json:"address"`
	Amount          string        `json:"amount"`
	ValidatorPubKey types.Pubkey  `json:"validator_pub_key"`
}

func (re *RewardEvent) Type() string {
	return TypeRewardEvent
}

func (re *RewardEvent) AddressString() string {
	return re.Address.String()
}

func (re *RewardEvent) address() types.Address {
	return re.Address
}

func (re *RewardEvent) ValidatorPubKeyString() string {
	return re.ValidatorPubKey.String()
}

func (re *RewardEvent) validatorPubKey() types.Pubkey {
	return re.ValidatorPubKey
}

func (re *RewardEvent) convert(pubKeyID uint16, addressID uint32) compact {
	result := new(reward)
	result.AddressID = addressID
	result.Role = NewRole(re.Role)
	bi, _ := big.NewInt(0).SetString(re.Amount, 10)
	result.Amount = bi.Bytes()
	result.PubKeyID = pubKeyID
	return result
}

type slash struct {
	AddressID uint32
	Amount    []byte
	Coin      uint32
	PubKeyID  uint16
}

func (s *slash) compile(pubKey [32]byte, address [20]byte) Event {
	event := new(SlashEvent)
	event.ValidatorPubKey = pubKey
	event.Address = address
	event.Coin = uint64(s.Coin)
	event.Amount = big.NewInt(0).SetBytes(s.Amount).String()
	return event
}

func (s *slash) addressID() uint32 {
	return s.AddressID
}

func (s *slash) pubKeyID() uint16 {
	return s.PubKeyID
}

type SlashEvent struct {
	Address         types.Address `json:"address"`
	Amount          string        `json:"amount"`
	Coin            uint64        `json:"coin"`
	ValidatorPubKey types.Pubkey  `json:"validator_pub_key"`
}

func (se *SlashEvent) Type() string {
	return TypeSlashEvent
}

func (se *SlashEvent) AddressString() string {
	return se.Address.String()
}

func (se *SlashEvent) address() types.Address {
	return se.Address
}

func (se *SlashEvent) ValidatorPubKeyString() string {
	return se.ValidatorPubKey.String()
}

func (se *SlashEvent) validatorPubKey() types.Pubkey {
	return se.ValidatorPubKey
}

func (se *SlashEvent) convert(pubKeyID uint16, addressID uint32) compact {
	result := new(slash)
	result.AddressID = addressID
	result.Coin = uint32(se.Coin)
	bi, _ := big.NewInt(0).SetString(se.Amount, 10)
	result.Amount = bi.Bytes()
	result.PubKeyID = pubKeyID
	return result
}

type unbond struct {
	AddressID uint32
	Amount    []byte
	Coin      uint32
	PubKeyID  uint16
}

func (u *unbond) compile(pubKey [32]byte, address [20]byte) Event {
	event := new(UnbondEvent)
	event.ValidatorPubKey = pubKey
	event.Address = address
	event.Coin = uint64(u.Coin)
	event.Amount = big.NewInt(0).SetBytes(u.Amount).String()
	return event
}

func (u *unbond) addressID() uint32 {
	return u.AddressID
}

func (u *unbond) pubKeyID() uint16 {
	return u.PubKeyID
}

type UnbondEvent struct {
	Address         types.Address `json:"address"`
	Amount          string        `json:"amount"`
	Coin            uint64        `json:"coin"`
	ValidatorPubKey types.Pubkey  `json:"validator_pub_key"`
}

func (ue *UnbondEvent) Type() string {
	return TypeUnbondEvent
}

func (ue *UnbondEvent) AddressString() string {
	return ue.Address.String()
}

func (ue *UnbondEvent) address() types.Address {
	return ue.Address
}

func (ue *UnbondEvent) ValidatorPubKeyString() string {
	return ue.ValidatorPubKey.String()
}

func (ue *UnbondEvent) validatorPubKey() types.Pubkey {
	return ue.ValidatorPubKey
}

func (ue *UnbondEvent) convert(pubKeyID uint16, addressID uint32) compact {
	result := new(unbond)
	result.AddressID = addressID
	result.Coin = uint32(ue.Coin)
	bi, _ := big.NewInt(0).SetString(ue.Amount, 10)
	result.Amount = bi.Bytes()
	result.PubKeyID = pubKeyID
	return result
}

type move struct {
	AddressID uint32
	Amount    []byte
	Coin      uint32
	PubKeyID  uint16
	WaitList  bool
}

func (u *move) compile(pubKey [32]byte, address [20]byte) Event {
	event := new(StakeMoveEvent)
	event.ValidatorPubKey = pubKey
	event.Address = address
	event.Coin = uint64(u.Coin)
	event.Amount = big.NewInt(0).SetBytes(u.Amount).String()
	event.WaitList = u.WaitList
	return event
}

func (u *move) addressID() uint32 {
	return u.AddressID
}

func (u *move) pubKeyID() uint16 {
	return u.PubKeyID
}

type StakeMoveEvent struct {
	Address         types.Address `json:"address"`
	Amount          string        `json:"amount"`
	Coin            uint64        `json:"coin"`
	ValidatorPubKey types.Pubkey  `json:"validator_pub_key"`
	WaitList        bool          `json:"waitlist"`
}

func (ue *StakeMoveEvent) Type() string {
	return TypeStakeMoveEvent
}

func (ue *StakeMoveEvent) AddressString() string {
	return ue.Address.String()
}

func (ue *StakeMoveEvent) address() types.Address {
	return ue.Address
}

func (ue *StakeMoveEvent) ValidatorPubKeyString() string {
	return ue.ValidatorPubKey.String()
}

func (ue *StakeMoveEvent) validatorPubKey() types.Pubkey {
	return ue.ValidatorPubKey
}

func (ue *StakeMoveEvent) convert(pubKeyID uint16, addressID uint32) compact {
	result := new(move)
	result.AddressID = addressID
	result.Coin = uint32(ue.Coin)
	bi, _ := big.NewInt(0).SetString(ue.Amount, 10)
	result.Amount = bi.Bytes()
	result.PubKeyID = pubKeyID
	result.WaitList = ue.WaitList
	return result
}

type kick struct {
	AddressID uint32
	Amount    []byte
	Coin      uint32
	PubKeyID  uint16
}

func (u *kick) compile(pubKey [32]byte, address [20]byte) Event {
	event := new(StakeKickEvent)
	event.ValidatorPubKey = pubKey
	event.Address = address
	event.Coin = uint64(u.Coin)
	event.Amount = big.NewInt(0).SetBytes(u.Amount).String()
	return event
}

func (u *kick) addressID() uint32 {
	return u.AddressID
}

func (u *kick) pubKeyID() uint16 {
	return u.PubKeyID
}

type StakeKickEvent struct {
	Address         types.Address `json:"address"`
	Amount          string        `json:"amount"`
	Coin            uint64        `json:"coin"`
	ValidatorPubKey types.Pubkey  `json:"validator_pub_key"`
}

func (ue *StakeKickEvent) Type() string {
	return TypeStakeKickEvent
}

func (ue *StakeKickEvent) AddressString() string {
	return ue.Address.String()
}

func (ue *StakeKickEvent) address() types.Address {
	return ue.Address
}

func (ue *StakeKickEvent) ValidatorPubKeyString() string {
	return ue.ValidatorPubKey.String()
}

func (ue *StakeKickEvent) validatorPubKey() types.Pubkey {
	return ue.ValidatorPubKey
}

func (ue *StakeKickEvent) convert(pubKeyID uint16, addressID uint32) compact {
	result := new(kick)
	result.AddressID = addressID
	result.Coin = uint32(ue.Coin)
	bi, _ := big.NewInt(0).SetString(ue.Amount, 10)
	result.Amount = bi.Bytes()
	result.PubKeyID = pubKeyID
	return result
}

type UpdateCommissionsEvent struct {
	Coin                    uint64 `json:"coin"`
	PayloadByte             string `json:"payload_byte"`
	Send                    string `json:"send"`
	BuyBancor               string `json:"buy_bancor"`
	SellBancor              string `json:"sell_bancor"`
	SellAllBancor           string `json:"sell_all_bancor"`
	BuyPool                 string `json:"buy_pool"`
	SellPool                string `json:"sell_pool"`
	SellAllPool             string `json:"sell_all_pool"`
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
	PriceVote               string `json:"price_vote"`
	EditCandidatePublicKey  string `json:"edit_candidate_public_key"`
	CreateSwapPool          string `json:"create_swap_pool"`
	AddLiquidity            string `json:"add_liquidity"`
	RemoveLiquidity         string `json:"remove_liquidity"`
	EditCandidateCommission string `json:"edit_candidate_commission"`
	MoveStake               string `json:"move_stake"`
	MintToken               string `json:"mint_token"`
	BurnToken               string `json:"burn_token"`
	VoteCommission          string `json:"vote_commission"`
	VoteUpdate              string `json:"vote_update"`
}

func (ce *UpdateCommissionsEvent) Type() string {
	return TypeUpdateCommissionsEvent
}

type UpdateNetworkEvent struct {
	Version string `json:"version"`
}

func (un *UpdateNetworkEvent) Type() string {
	return TypeUpdateNetworkEvent
}
