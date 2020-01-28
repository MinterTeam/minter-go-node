package types11

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"strings"
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

func (bA *BitArray) getIndex(i uint) bool {
	if i >= bA.Bits {
		return false
	}
	return bA.Elems[i/64]&(uint64(1)<<uint(i%64)) > 0
}

// String returns a string representation of BitArray: BA{<bit-string>},
// where <bit-string> is a sequence of 'x' (1) and '_' (0).
// The <bit-string> includes spaces and newlines to help people.
// For a simple sequence of 'x' and '_' characters with no spaces or newlines,
// see the MarshalJSON() method.
// Example: "BA{_x_}" or "nil-BitArray" for nil.
func (bA *BitArray) String() string {
	return bA.StringIndented("")
}

// StringIndented returns the same thing as String(), but applies the indent
// at every 10th bit, and twice at every 50th bit.
func (bA *BitArray) StringIndented(indent string) string {
	if bA == nil {
		return "nil-BitArray"
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	return bA.stringIndented(indent)
}

func (bA *BitArray) stringIndented(indent string) string {
	lines := []string{}
	bits := ""
	for i := uint(0); i < bA.Bits; i++ {
		if bA.getIndex(i) {
			bits += "x"
		} else {
			bits += "_"
		}
		if i%100 == 99 {
			lines = append(lines, bits)
			bits = ""
		}
		if i%10 == 9 {
			bits += indent
		}
		if i%50 == 49 {
			bits += indent
		}
	}
	if len(bits) > 0 {
		lines = append(lines, bits)
	}
	return fmt.Sprintf("BA{%v:%v}", bA.Bits, strings.Join(lines, indent))
}

// Bytes returns the byte representation of the bits within the bitarray.
func (bA *BitArray) Bytes() []byte {
	bA.mtx.Lock()
	defer bA.mtx.Unlock()

	numBytes := (bA.Bits + 7) / 8
	bytes := make([]byte, numBytes)
	for i := 0; i < len(bA.Elems); i++ {
		elemBytes := [8]byte{}
		binary.LittleEndian.PutUint64(elemBytes[:], bA.Elems[i])
		copy(bytes[i*8:], elemBytes[:])
	}
	return bytes
}

// MarshalJSON implements json.Marshaler interface by marshaling bit array
// using a custom format: a string of '-' or 'x' where 'x' denotes the 1 bit.
func (bA *BitArray) MarshalJSON() ([]byte, error) {
	if bA == nil {
		return []byte("null"), nil
	}

	bA.mtx.Lock()
	defer bA.mtx.Unlock()

	bits := `"`
	for i := uint(0); i < bA.Bits; i++ {
		if bA.getIndex(i) {
			bits += `x`
		} else {
			bits += `_`
		}
	}
	bits += `"`
	return []byte(bits), nil
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

const PubKeyLength = 32

type Pubkey [PubKeyLength]byte

func BytesToPubkey(b []byte) Pubkey {
	var p Pubkey
	p.SetBytes(b)
	return p
}

func (p *Pubkey) SetBytes(b []byte) {
	if len(b) > len(p) {
		b = b[len(b)-PubKeyLength:]
	}
	copy(p[PubKeyLength-len(b):], b)
}

func (p Pubkey) Bytes() []byte { return p[:] }

func (p Pubkey) String() string {
	return fmt.Sprintf("Mp%x", p[:])
}

func (p Pubkey) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p Pubkey) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", p.String())), nil
}

func (p *Pubkey) UnmarshalJSON(input []byte) error {
	b, err := hex.DecodeString(string(input)[3 : len(input)-1])
	copy(p[:], b)

	return err
}
