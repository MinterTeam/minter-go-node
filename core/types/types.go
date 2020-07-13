package types

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"math/big"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
)

const (
	HashLength       = 32
	AddressLength    = 20
	PubKeyLength     = 32
	CoinSymbolLength = 10
	BasecoinID       = 0
)

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})
)

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}
func StringToHash(s string) Hash { return BytesToHash([]byte(s)) }
func BigToHash(b *big.Int) Hash  { return BytesToHash(b.Bytes()) }
func HexToHash(s string) Hash    { return BytesToHash(FromHex(s, "Mh")) }

// Get the string representation of the underlying hash
func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h Hash) Hex() string   { return hexutil.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h[:3], h[29:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (h Hash) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), h[:])
}

// UnmarshalText parses a hash in hex syntax.
func (h *Hash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Hash", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(hashT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// Sets the hash to the value of b. If b is larger than len(h), 'b' will be cropped (from the left).
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// Set string `s` to h. If s is larger than len(h) s will be cropped (from left) to fit.
func (h *Hash) SetString(s string) { h.SetBytes([]byte(s)) }

// Sets h to other
func (h *Hash) Set(other Hash) {
	for i, v := range other {
		h[i] = v
	}
}

// Generate implements testing/quick.Generator.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))
	for i := len(h) - 1; i > m; i-- {
		h[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

func EmptyHash(h Hash) bool {
	return h == Hash{}
}

// UnprefixedHash allows marshaling a Hash without 0x prefix.
type UnprefixedHash Hash

// UnmarshalText decodes the hash from hex. The 0x prefix is optional.
func (h *UnprefixedHash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedHash", input, h[:])
}

// MarshalText encodes the hash as hex.
func (h UnprefixedHash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

/////////// Coin

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

func (c CoinSymbol) Compare(c2 CoinSymbol) int {
	return bytes.Compare(c.Bytes(), c2.Bytes())
}

func (c CoinSymbol) IsBaseCoin() bool {
	return c.Compare(GetBaseCoin()) == 0
}

func (c CoinSymbol) GetBaseSymbol() CoinSymbol {
	return StrToCoinSymbol(strings.Split(c.String(), "-")[0])
}

func (c CoinSymbol) GetVersion() uint16 {
	parts := strings.Split(c.String(), "-")
	if len(parts) == 1 {
		return 0
	}

	return 1
}

func StrToCoinSymbol(s string) CoinSymbol {
	var symbol CoinSymbol
	copy(symbol[:], []byte(s))
	return symbol
}

type CoinID uint32

func (c CoinID) IsBaseCoin() bool {
	return c == GetBaseCoinID()
}

func (c CoinID) String() string {
	return strconv.Itoa(int(c))
}

func (c CoinID) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, c.Uint32())
	return b
}

func (c CoinID) Uint32() uint32 {
	return uint32(c)
}

type CoinVersion = uint16

/////////// Address

type Address [AddressLength]byte

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}
func StringToAddress(s string) Address { return BytesToAddress([]byte(s)) }
func BigToAddress(b *big.Int) Address  { return BytesToAddress(b.Bytes()) }
func HexToAddress(s string) Address    { return BytesToAddress(FromHex(s, "Mx")) }

// IsHexAddress verifies whether a string can represent a valid hex-encoded
// Minter address or not.
func IsHexAddress(s string) bool {
	if hasHexPrefix(s, "Mx") {
		s = s[2:]
	}
	return len(s) == 2*AddressLength && isHex(s)
}

// Get the string representation of the underlying address
func (a Address) Str() string   { return string(a[:]) }
func (a Address) Bytes() []byte { return a[:] }
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }
func (a Address) Hash() Hash    { return BytesToHash(a[:]) }

func (a Address) Hex() string {
	return "Mx" + hex.EncodeToString(a[:])
}

// String implements the stringer interface and is used also by the logger.
func (a Address) String() string {
	return a.Hex()
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (a Address) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), a[:])
}

// Sets the address to the value of b. If b is larger than len(a) it will panic
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// Set string `s` to a. If s is larger than len(a) it will panic
func (a *Address) SetString(s string) { a.SetBytes([]byte(s)) }

// Sets a to other
func (a *Address) Set(other Address) {
	for i, v := range other {
		a[i] = v
	}
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

// UnmarshalJSON parses a hash in hex syntax.
func (a *Address) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(addressT, input, a[:])
}

func (a *Address) Compare(a2 Address) int {
	return bytes.Compare(a.Bytes(), a2.Bytes())
}

// UnprefixedHash allows marshaling an Address without 0x prefix.
type UnprefixedAddress Address

// UnmarshalText decodes the address from hex. The 0x prefix is optional.
func (a *UnprefixedAddress) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedAddress", input, a[:])
}

// MarshalText encodes the address as hex.
func (a UnprefixedAddress) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(a[:])), nil
}

type Pubkey [32]byte

func HexToPubkey(s string) Pubkey { return BytesToPubkey(FromHex(s, "Mp")) }

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

func (p Pubkey) Equals(p2 Pubkey) bool {
	return p == p2
}

type TmAddress [20]byte
