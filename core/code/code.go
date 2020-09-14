package code

import (
	"strconv"
)

// Codes for transaction checks and delivers responses
const (
	// general
	OK                           uint32 = 0
	WrongNonce                   uint32 = 101
	CoinNotExists                uint32 = 102
	CoinReserveNotSufficient     uint32 = 103
	TxTooLarge                   uint32 = 105
	DecodeError                  uint32 = 106
	InsufficientFunds            uint32 = 107
	TxPayloadTooLarge            uint32 = 109
	TxServiceDataTooLarge        uint32 = 110
	InvalidMultisendData         uint32 = 111
	CoinSupplyOverflow           uint32 = 112
	TxFromSenderAlreadyInMempool uint32 = 113
	TooLowGasPrice               uint32 = 114
	WrongChainID                 uint32 = 115
	CoinReserveUnderflow         uint32 = 116
	WrongHaltHeight              uint32 = 117

	// coin creation
	CoinAlreadyExists uint32 = 201
	WrongCrr          uint32 = 202
	InvalidCoinSymbol uint32 = 203
	InvalidCoinName   uint32 = 204
	WrongCoinSupply   uint32 = 205

	// recreate coin
	IsNotOwnerOfCoin uint32 = 206

	// convert
	CrossConvert              uint32 = 301
	MaximumValueToSellReached uint32 = 302
	MinimumValueToBuyReached  uint32 = 303

	// candidate
	CandidateExists       uint32 = 401
	WrongCommission       uint32 = 402
	CandidateNotFound     uint32 = 403
	StakeNotFound         uint32 = 404
	InsufficientStake     uint32 = 405
	IsNotOwnerOfCandidate uint32 = 406
	IncorrectPubKey       uint32 = 407
	StakeShouldBePositive uint32 = 408
	TooLowStake           uint32 = 409
	PublicKeyInBlockList  uint32 = 410
	NewPublicKeyIsBad     uint32 = 411
	InsufficientWaitList  uint32 = 412

	// check
	CheckInvalidLock uint32 = 501
	CheckExpired     uint32 = 502
	CheckUsed        uint32 = 503
	TooHighGasPrice  uint32 = 504
	WrongGasCoin     uint32 = 505
	TooLongNonce     uint32 = 506

	// multisig
	IncorrectWeights        uint32 = 601
	MultisigExists          uint32 = 602
	MultisigNotExists       uint32 = 603
	IncorrectMultiSignature uint32 = 604
	TooLargeOwnersList      uint32 = 605
	DuplicatedAddresses     uint32 = 606
)

type wrongNonce struct {
	Code          string `json:"code,omitempty"`
	ExpectedNonce string `json:"expected_nonce,omitempty"`
	GotNonce      string `json:"got_nonce,omitempty"`
}

func NewWrongNonce(expectedNonce string, gotNonce string) *wrongNonce {
	return &wrongNonce{Code: strconv.Itoa(int(WrongNonce)), ExpectedNonce: expectedNonce, GotNonce: gotNonce}
}

type coinNotExists struct {
	Code       string `json:"code,omitempty"`
	CoinSymbol string `json:"coin_symbol,omitempty"`
	CoinId     string `json:"coin_id,omitempty"`
}

func NewCoinNotExists(coinSymbol string, coinId string) *coinNotExists {
	return &coinNotExists{Code: strconv.Itoa(int(CoinNotExists)), CoinSymbol: coinSymbol, CoinId: coinId}
}

type coinReserveNotSufficient struct {
	Code             string `json:"code,omitempty"`
	CoinSymbol       string `json:"coin_symbol,omitempty"`
	CoinId           string `json:"coin_id,omitempty"`
	HasBipValue      string `json:"has_bip_value,omitempty"`
	RequiredBipValue string `json:"required_bip_value,omitempty"`
}

func NewCoinReserveNotSufficient(coinSymbol string, coinId string, hasBipValue string, requiredBipValue string) *coinReserveNotSufficient {
	return &coinReserveNotSufficient{Code: strconv.Itoa(int(CoinReserveNotSufficient)), CoinSymbol: coinSymbol, CoinId: coinId, HasBipValue: hasBipValue, RequiredBipValue: requiredBipValue}
}

type txTooLarge struct {
	Code        string `json:"code,omitempty"`
	MaxTxLength string `json:"max_tx_length,omitempty"`
	GotTxLength string `json:"got_tx_length,omitempty"`
}

func NewTxTooLarge(maxTxLength string, gotTxLength string) *txTooLarge {
	return &txTooLarge{Code: strconv.Itoa(int(TxTooLarge)), MaxTxLength: maxTxLength, GotTxLength: gotTxLength}
}

type decodeError struct {
	Code string `json:"code,omitempty"`
}

func NewDecodeError() *decodeError {
	return &decodeError{Code: strconv.Itoa(int(DecodeError))}
}

type insufficientFunds struct {
	Code           string `json:"code,omitempty"`
	Sender         string `json:"sender,omitempty"`
	NeededBipValue string `json:"needed_bip_value,omitempty"`
	CoinSymbol     string `json:"coin_symbol,omitempty"`
	CoinId         string `json:"coin_id,omitempty"`
}

func NewInsufficientFunds(sender string, neededBipValue string, coinSymbol string, coinId string) *insufficientFunds {
	return &insufficientFunds{Code: strconv.Itoa(int(InsufficientFunds)), Sender: sender, NeededBipValue: neededBipValue, CoinSymbol: coinSymbol, CoinId: coinId}
}

type txPayloadTooLarge struct {
	Code             string `json:"code,omitempty"`
	MaxPayloadLength string `json:"max_payload_length,omitempty"`
	GotPayloadLength string `json:"got_payload_length,omitempty"`
}

func NewTxPayloadTooLarge(maxPayloadLength string, gotPayloadLength string) *txPayloadTooLarge {
	return &txPayloadTooLarge{Code: strconv.Itoa(int(TxPayloadTooLarge)), MaxPayloadLength: maxPayloadLength, GotPayloadLength: gotPayloadLength}
}

type txServiceDataTooLarge struct {
	Code                 string `json:"code,omitempty"`
	MaxServiceDataLength string `json:"max_service_data_length,omitempty"`
	GotServiceDataLength string `json:"got_service_data_length,omitempty"`
}

func NewTxServiceDataTooLarge(maxServiceDataLength string, gotServiceDataLength string) *txServiceDataTooLarge {
	return &txServiceDataTooLarge{Code: strconv.Itoa(int(TxServiceDataTooLarge)), MaxServiceDataLength: maxServiceDataLength, GotServiceDataLength: gotServiceDataLength}
}

type invalidMultisendData struct {
	Code        string `json:"code,omitempty"`
	MinQuantity string `json:"min_quantity,omitempty"`
	MaxQuantity string `json:"max_quantity,omitempty"`
	GotQuantity string `json:"got_quantity,omitempty"`
}

func NewInvalidMultisendData(minQuantity string, maxQuantity string, gotQuantity string) *invalidMultisendData {
	return &invalidMultisendData{Code: strconv.Itoa(int(InvalidMultisendData)), MinQuantity: minQuantity, MaxQuantity: maxQuantity, GotQuantity: gotQuantity}
}

type coinSupplyOverflow struct {
	Code          string `json:"code,omitempty"`
	Delta         string `json:"delta,omitempty"`
	CoinSupply    string `json:"coin_supply,omitempty"`
	CurrentSupply string `json:"current_supply,omitempty"`
	MaxCoinSupply string `json:"max_coin_supply,omitempty"`
	CoinSymbol    string `json:"coin_symbol,omitempty"`
	CoinId        string `json:"coin_id,omitempty"`
}

func NewCoinSupplyOverflow(delta string, coinSupply string, currentSupply string, maxCoinSupply string, coinSymbol string, coinId string) *coinSupplyOverflow {
	return &coinSupplyOverflow{Code: strconv.Itoa(int(CoinSupplyOverflow)), Delta: delta, CoinSupply: coinSupply, CurrentSupply: currentSupply, MaxCoinSupply: maxCoinSupply, CoinSymbol: coinSymbol, CoinId: coinId}
}

type txFromSenderAlreadyInMempool struct {
	Code   string `json:"code,omitempty"`
	Sender string `json:"sender,omitempty"`
}

func NewTxFromSenderAlreadyInMempool(sender string, itoa string) *txFromSenderAlreadyInMempool {
	return &txFromSenderAlreadyInMempool{Code: strconv.Itoa(int(TxFromSenderAlreadyInMempool)), Sender: sender}
}

type tooLowGasPrice struct {
	Code        string `json:"code,omitempty"`
	MinGasPrice string `json:"min_gas_price,omitempty"`
	GotGasPrice string `json:"got_gas_price,omitempty"`
}

func NewTooLowGasPrice(minGasPrice string, gotGasPrice string) *tooLowGasPrice {
	return &tooLowGasPrice{Code: strconv.Itoa(int(TooLowGasPrice)), MinGasPrice: minGasPrice, GotGasPrice: gotGasPrice}
}

type wrongChainID struct {
	Code           string `json:"code,omitempty"`
	CurrentChainId string `json:"current_chain_id,omitempty"`
	GotChainId     string `json:"got_chain_id,omitempty"`
}

func NewWrongChainID(currentChainId string, gotChainId string) *wrongChainID {
	return &wrongChainID{Code: strconv.Itoa(int(WrongChainID)), CurrentChainId: currentChainId, GotChainId: gotChainId}
}

type coinReserveUnderflow struct {
	Code           string `json:"code,omitempty"`
	Delta          string `json:"delta,omitempty"`
	CoinReserve    string `json:"coin_reserve,omitempty"`
	CurrentReserve string `json:"current_reserve,omitempty"`
	MinCoinReserve string `json:"min_coin_reserve,omitempty"`
	CoinSymbol     string `json:"coin_symbol,omitempty"`
	CoinId         string `json:"coin_id,omitempty"`
}

func NewCoinReserveUnderflow(delta string, coinReserve string, currentReserve string, minCoinReserve string, coinSymbol string, coinId string) *coinReserveUnderflow {
	return &coinReserveUnderflow{Code: strconv.Itoa(int(CoinReserveUnderflow)), Delta: delta, CoinReserve: coinReserve, CurrentReserve: currentReserve, MinCoinReserve: minCoinReserve, CoinSymbol: coinSymbol, CoinId: coinId}
}

type coinAlreadyExists struct {
	Code       string `json:"code,omitempty"`
	CoinSymbol string `json:"coin_symbol,omitempty"`
	CoinId     string `json:"coin_id,omitempty"`
}

func NewCoinAlreadyExists(coinSymbol string, coinId string) *coinAlreadyExists {
	return &coinAlreadyExists{Code: strconv.Itoa(int(CoinAlreadyExists)), CoinSymbol: coinSymbol, CoinId: coinId}
}

type invalidCoinSymbol struct {
	Code       string `json:"code,omitempty"`
	Pattern    string `json:"pattern,omitempty"`
	CoinSymbol string `json:"coin_symbol,omitempty"`
}

func NewInvalidCoinSymbol(pattern string, coinSymbol string) *invalidCoinSymbol {
	return &invalidCoinSymbol{Code: strconv.Itoa(int(InvalidCoinSymbol)), Pattern: pattern, CoinSymbol: coinSymbol}
}

type invalidCoinName struct {
	Code     string `json:"code,omitempty"`
	MaxBytes string `json:"max_bytes,omitempty"`
	GotBytes string `json:"got_bytes,omitempty"`
}

func NewInvalidCoinName(maxBytes string, gotBytes string) *invalidCoinName {
	return &invalidCoinName{Code: strconv.Itoa(int(InvalidCoinName)), MaxBytes: maxBytes, GotBytes: gotBytes}
}

// todo
type wrongCoinSupply struct {
	Code     string `json:"code,omitempty"`
	MaxBytes string `json:"max_bytes,omitempty"`
	GotBytes string `json:"got_bytes,omitempty"`
}

func NewWrongCoinSupply(maxBytes string, gotBytes string) *wrongCoinSupply {
	return &wrongCoinSupply{Code: strconv.Itoa(int(WrongCoinSupply)), MaxBytes: maxBytes, GotBytes: gotBytes}
}

type tooHighGasPrice struct {
	Code string `json:"code,omitempty"`
}

func NewTooHighGasPrice() *tooHighGasPrice {
	return &tooHighGasPrice{Code: strconv.Itoa(int(TooHighGasPrice))}
}
