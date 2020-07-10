package events

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/go-amino"
	"math/big"
)

func RegisterAminoEvents(codec *amino.Codec) {
	codec.RegisterInterface((*Event)(nil), nil)
	codec.RegisterConcrete(RewardEvent{},
		"minter/RewardEvent", nil)
	codec.RegisterConcrete(SlashEvent{},
		"minter/SlashEvent", nil)
	codec.RegisterConcrete(UnbondEvent{},
		"minter/UnbondEvent", nil)
}

type Event interface {
	address() types.Address
	validatorPubKey() types.Pubkey
	AddressString() string
	ValidatorPubKeyString() string
	convert(pubKeyID uint16, addressID uint32) compactEvent
}

type compactEvent interface {
	compile(pubKey string, address [20]byte) Event
	addressID() uint32
	pubKeyID() uint16
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

func (r *reward) compile(pubKey string, address [20]byte) Event {
	event := new(RewardEvent)
	copy(event.ValidatorPubKey[:], pubKey)
	copy(event.Address[:], address[:])
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

func (re *RewardEvent) convert(pubKeyID uint16, addressID uint32) compactEvent {
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
	Coin      [10]byte
	PubKeyID  uint16
}

func (s *slash) compile(pubKey string, address [20]byte) Event {
	event := new(SlashEvent)
	copy(event.ValidatorPubKey[:], pubKey)
	copy(event.Address[:], address[:])
	copy(event.Coin.Bytes(), s.Coin[:])
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
	Address         types.Address    `json:"address"`
	Amount          string           `json:"amount"`
	Coin            types.CoinID     `json:"coin"`
	ValidatorPubKey types.Pubkey     `json:"validator_pub_key"`
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

func (se *SlashEvent) convert(pubKeyID uint16, addressID uint32) compactEvent {
	result := new(slash)
	result.AddressID = addressID
	copy(result.Coin[:], se.Coin.Bytes())
	bi, _ := big.NewInt(0).SetString(se.Amount, 10)
	result.Amount = bi.Bytes()
	result.PubKeyID = pubKeyID
	return result
}

type unbond struct {
	AddressID uint32
	Amount    []byte
	Coin      [10]byte
	PubKeyID  uint16
}

func (u *unbond) compile(pubKey string, address [20]byte) Event {
	event := new(UnbondEvent)
	copy(event.ValidatorPubKey[:], pubKey)
	copy(event.Address[:], address[:])
	copy(event.Coin.Bytes(), u.Coin[:])
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
	Address         types.Address    `json:"address"`
	Amount          string           `json:"amount"`
	Coin            types.CoinID     `json:"coin"`
	ValidatorPubKey types.Pubkey     `json:"validator_pub_key"`
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

func (ue *UnbondEvent) convert(pubKeyID uint16, addressID uint32) compactEvent {
	result := new(unbond)
	result.AddressID = addressID
	copy(result.Coin[:], ue.Coin.Bytes())
	bi, _ := big.NewInt(0).SetString(ue.Amount, 10)
	result.Amount = bi.Bytes()
	result.PubKeyID = pubKeyID
	return result
}
