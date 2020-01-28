package types11

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"sync"
)

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

type Pubkey [32]byte

const CoinSymbolLength = 10

type CoinSymbol [CoinSymbolLength]byte

func (c CoinSymbol) String() string { return string(bytes.Trim(c[:], "\x00")) }
func (c CoinSymbol) Bytes() []byte  { return c[:] }

func (c CoinSymbol) MarshalJSON() ([]byte, error) {

	buffer := bytes.NewBufferString("\"")
	buffer.WriteString(c.String())
	buffer.WriteString("\"")

	return buffer.Bytes(), nil
}

func (c *CoinSymbol) UnmarshalJSON(input []byte) error {
	*c = StrToCoinSymbol(string(input[1 : len(input)-1]))
	return nil
}

func StrToCoinSymbol(s string) CoinSymbol {
	var symbol CoinSymbol
	copy(symbol[:], []byte(s))
	return symbol
}

type BitArray struct {
	mtx   sync.Mutex
	Bits  uint     `json:"bits"`  // NOTE: persisted via reflect, must be exported
	Elems []uint64 `json:"elems"` // NOTE: persisted via reflect, must be exported
}

const AddressLength = 20

type Address [AddressLength]byte

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}
func (a Address) String() string {
	return a.Hex()
}
func (a Address) Hex() string {
	return "Mx" + hex.EncodeToString(a[:])
}

// MarshalText returns the hex representation of a.
func (a Address) MarshalText() ([]byte, error) {
	return hexutil.Bytes(a[:]).MarshalText()
}

// UnmarshalText parses a hash in hex syntax.
func (a *Address) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Address", input, a[:])
}

func (a *Address) Unmarshal(input []byte) error {
	copy(a[:], input)
	return nil
}

func (a Address) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", a.String())), nil
}
