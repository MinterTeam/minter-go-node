package types

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
)

// Types lengths
const (
	HashLength              = 32
	AddressLength           = 20
	PubKeyLength            = 32
	CoinSymbolLength        = 10
	TendermintAddressLength = 20
)

const (
	// BasecoinID is an ID of a base coin
	BasecoinID = 0
)

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})
)

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

// BytesToHash converts given byte slice to Hash
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// Str returns the string representation of the underlying hash
func (h Hash) Str() string { return string(h[:]) }

// Bytes returns the bytes representation of the underlying hash
func (h Hash) Bytes() []byte { return h[:] }

// Big returns the big.Int representation of the underlying hash
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex returns the hex-string representation of the underlying hash
func (h Hash) Hex() string { return hexutil.Encode(h[:]) }

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

// SetBytes Sets the hash to the value of b. If b is larger than len(h), 'b' will be cropped (from the left).
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// SetString sets string `s` to h. If s is larger than len(h) s will be cropped (from left) to fit.
func (h *Hash) SetString(s string) { h.SetBytes([]byte(s)) }

// Set h to other
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

// EmptyHash checks if given Hash is empty
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

// ///////// Coin

// CoinSymbol represents the 10 byte coin symbol.
type CoinSymbol [CoinSymbolLength]byte

func (c CoinSymbol) String() string { return string(bytes.Trim(c[:], "\x00")) }

// Bytes returns the bytes representation of the underlying CoinSymbol
func (c CoinSymbol) Bytes() []byte { return c[:] }

// MarshalJSON encodes coin to json
func (c CoinSymbol) MarshalJSON() ([]byte, error) {

	buffer := bytes.NewBufferString("\"")
	buffer.WriteString(c.String())
	buffer.WriteString("\"")

	return buffer.Bytes(), nil
}

// UnmarshalJSON parses a coinSymbol from json
func (c *CoinSymbol) UnmarshalJSON(input []byte) error {
	*c = StrToCoinSymbol(string(input[1 : len(input)-1]))
	return nil
}

// Compare compares coin symbols.
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func (c CoinSymbol) Compare(c2 CoinSymbol) int {
	return bytes.Compare(c.Bytes(), c2.Bytes())
}

// IsBaseCoin checks if coin is a base coin
func (c CoinSymbol) IsBaseCoin() bool {
	return c.Compare(GetBaseCoin()) == 0
}

// StrToCoinSymbol converts given string to a coin symbol
func StrToCoinSymbol(s string) CoinSymbol {
	var symbol CoinSymbol
	copy(symbol[:], s)
	return symbol
}

// StrToCoinBaseSymbol converts give string to a coin base symbol
func StrToCoinBaseSymbol(s string) CoinSymbol {
	delimiter := strings.Index(s, "-")
	if delimiter != -1 {
		return StrToCoinSymbol(s[:delimiter])
	}

	return StrToCoinSymbol(s)
}

// GetVersionFromSymbol returns coin version extracted from symbol
func GetVersionFromSymbol(s string) CoinVersion {
	parts := strings.Split(s, "-")
	if len(parts) == 1 {
		return 0
	}

	v, _ := strconv.ParseUint(parts[1], 10, 16)
	return CoinVersion(v)
}

// CoinID represents coin id
type CoinID uint32

// IsBaseCoin checks if
func (c CoinID) IsBaseCoin() bool {
	return c == GetBaseCoinID()
}

func (c CoinID) String() string {
	return strconv.Itoa(int(c))
}

// Bytes returns LittleEndian encoded bytes of coin id
func (c CoinID) Bytes() []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, c.Uint32())
	return b
}

// Uint32 returns coin id as uint32
func (c CoinID) Uint32() uint32 {
	return uint32(c)
}

// BytesToCoinID converts bytes to coin id. Expects LittleEndian encoding.
func BytesToCoinID(bytes []byte) CoinID {
	return CoinID(binary.LittleEndian.Uint32(bytes))
}

// CoinVersion represents coin version info
type CoinVersion = uint16

// ///////// Address

// Address represents 20-byte address in Minter Blockchain
type Address [AddressLength]byte

// BytesToAddress converts given byte slice to Address
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// StringToAddress converts given string to Address
func StringToAddress(s string) Address { return BytesToAddress([]byte(s)) }

// BigToAddress converts given big.Int to Address
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress converts given hex string to Address
func HexToAddress(s string) Address { return BytesToAddress(FromHex(s, "Mx")) }

// IsHexAddress verifies whether a string can represent a valid hex-encoded
// Minter address or not.
func IsHexAddress(s string) bool {
	if hasHexPrefix(s, "Mx") {
		s = s[2:]
	}
	return len(s) == 2*AddressLength && isHex(s)
}

// Str returns the string representation of the underlying address
func (a Address) Str() string { return string(a[:]) }

// Bytes returns the byte representation of the underlying address
func (a Address) Bytes() []byte { return a[:] }

// Big returns the big.Int representation of the underlying address
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }

// Hash returns the Hash representation of the underlying address
func (a Address) Hash() Hash { return BytesToHash(a[:]) }

// Hex returns the hex-string representation of the underlying address
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

// SetBytes Sets the address to the value of b. If b is larger than len(a) it will panic
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// SetString set string `s` to a. If s is larger than len(a) it will panic
func (a *Address) SetString(s string) { a.SetBytes([]byte(s)) }

// Set Sets a to other
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

// Unmarshal parses a hash from byte slice.
func (a *Address) Unmarshal(input []byte) error {
	copy(a[:], input)
	return nil
}

// MarshalJSON marshals given address to json format.
func (a Address) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", a.String())), nil
}

// UnmarshalJSON parses a hash in hex syntax.
func (a *Address) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(addressT, input, a[:])
}

// Compare compares addresses.
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func (a *Address) Compare(a2 Address) int {
	return bytes.Compare(a.Bytes(), a2.Bytes())
}

// Pubkey represents 32 byte public key of a validator
type Pubkey [PubKeyLength]byte

// HexToPubkey decodes given string into Pubkey
func HexToPubkey(s string) Pubkey { return BytesToPubkey(FromHex(s, "Mp")) }

// BytesToPubkey decodes given bytes into Pubkey
func BytesToPubkey(b []byte) Pubkey {
	var p Pubkey
	p.SetBytes(b)
	return p
}

// SetBytes sets given bytes as public key
func (p *Pubkey) SetBytes(b []byte) {
	if len(b) > len(p) {
		b = b[len(b)-PubKeyLength:]
	}
	copy(p[PubKeyLength-len(b):], b)
}

// Bytes returns underlying bytes
func (p Pubkey) Bytes() []byte { return p[:] }

func (p Pubkey) String() string {
	return fmt.Sprintf("Mp%x", p[:])
}

// MarshalText encodes Pubkey from to text.
func (p Pubkey) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

// MarshalJSON encodes Pubkey from to json format.
func (p Pubkey) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", p.String())), nil
}

// UnmarshalJSON decodes Pubkey from json format.
func (p *Pubkey) UnmarshalJSON(input []byte) error {
	b, err := hex.DecodeString(string(input)[3 : len(input)-1])
	copy(p[:], b)

	return err
}

// Equals checks if public keys are equal
func (p Pubkey) Equals(p2 Pubkey) bool {
	return p == p2
}

// TmAddress represents Tendermint address
type TmAddress [TendermintAddressLength]byte

func GetTmAddress(publicKey Pubkey) TmAddress {
	// set tm address
	var pubkey ed25519.PubKey
	copy(pubkey[:], publicKey[:])

	var address TmAddress
	copy(address[:], pubkey.Address().Bytes())

	return address
}
