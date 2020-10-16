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
	HaltAlreadyExists            uint32 = 118

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
	IncorrectWeights                  uint32 = 601
	MultisigExists                    uint32 = 602
	MultisigNotExists                 uint32 = 603
	IncorrectMultiSignature           uint32 = 604
	TooLargeOwnersList                uint32 = 605
	DuplicatedAddresses               uint32 = 606
	DifferentCountAddressesAndWeights uint32 = 607
	IncorrectTotalWeights             uint32 = 608
	NotEnoughMultisigVotes            uint32 = 609
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

type wrongGasCoin struct {
	Code               string `json:"code,omitempty"`
	TxGasCoinSymbol    string `json:"tx_coin_symbol,omitempty"`
	TxGasCoinId        string `json:"tx_coin_id,omitempty"`
	CheckGasCoinSymbol string `json:"check_coin_symbol,omitempty"`
	CheckGasCoinId     string `json:"check_coin_id,omitempty"`
}

func NewWrongGasCoin(txCoinSymbol string, txCoinId string, checkGasCoinSymbol, checkGasCoinId string) *wrongGasCoin {
	return &wrongGasCoin{Code: strconv.Itoa(int(WrongGasCoin)), TxGasCoinSymbol: txCoinSymbol, TxGasCoinId: txCoinId, CheckGasCoinId: checkGasCoinId, CheckGasCoinSymbol: checkGasCoinSymbol}
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
	Code        string `json:"code,omitempty"`
	Sender      string `json:"sender,omitempty"`
	BlockHeight string `json:"block_height,omitempty"`
}

func NewTxFromSenderAlreadyInMempool(sender string, block string) *txFromSenderAlreadyInMempool {
	return &txFromSenderAlreadyInMempool{Code: strconv.Itoa(int(TxFromSenderAlreadyInMempool)), Sender: sender, BlockHeight: block}
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

type tooHighGasPrice struct {
	Code             string `json:"code,omitempty"`
	MaxCheckGasPrice string `json:"max_check_gas_price,omitempty"`
	CurrentGasPrice  string `json:"current_gas_price,omitempty"`
}

func NewTooHighGasPrice(maxCheckGasPrice, currentGasPrice string) *tooHighGasPrice {
	return &tooHighGasPrice{Code: strconv.Itoa(int(TooHighGasPrice)), MaxCheckGasPrice: maxCheckGasPrice, CurrentGasPrice: currentGasPrice}
}

type candidateExists struct {
	Code      string `json:"code,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
}

func NewCandidateExists(publicKey string) *candidateExists {
	return &candidateExists{Code: strconv.Itoa(int(CandidateExists)), PublicKey: publicKey}
}

type candidateNotFound struct {
	Code      string `json:"code,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
}

func NewCandidateNotFound(publicKey string) *candidateNotFound {
	return &candidateNotFound{Code: strconv.Itoa(int(CandidateNotFound)), PublicKey: publicKey}
}

type publicKeyInBlockList struct {
	Code      string `json:"code,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
}

func NewPublicKeyInBlockList(publicKey string) *publicKeyInBlockList {
	return &publicKeyInBlockList{Code: strconv.Itoa(int(PublicKeyInBlockList)), PublicKey: publicKey}
}

type newPublicKeyIsBad struct {
	Code         string `json:"code,omitempty"`
	PublicKey    string `json:"public_key,omitempty"`
	NewPublicKey string `json:"new_public_key,omitempty"`
}

func NewNewPublicKeyIsBad(publicKey, newPublicKey string) *newPublicKeyIsBad {
	return &newPublicKeyIsBad{Code: strconv.Itoa(int(NewPublicKeyIsBad)), PublicKey: publicKey, NewPublicKey: newPublicKey}
}

type insufficientWaitList struct {
	Code          string `json:"code,omitempty"`
	WaitlistValue string `json:"waitlist_value,omitempty"`
	NeededValue   string `json:"needed_value,omitempty"`
}

func NewInsufficientWaitList(waitlistValue, neededValue string) *insufficientWaitList {
	return &insufficientWaitList{Code: strconv.Itoa(int(InsufficientWaitList)), WaitlistValue: waitlistValue, NeededValue: neededValue}
}

type stakeNotFound struct {
	Code       string `json:"code,omitempty"`
	PublicKey  string `json:"public_key,omitempty"`
	Owner      string `json:"owner,omitempty"`
	CoinSymbol string `json:"coin_symbol,omitempty"`
	CoinId     string `json:"coin_id,omitempty"`
}

func NewStakeNotFound(publicKey string, owner string, coinId string, coinSymbol string) *stakeNotFound {
	return &stakeNotFound{Code: strconv.Itoa(int(StakeNotFound)), PublicKey: publicKey, Owner: owner, CoinId: coinId, CoinSymbol: coinSymbol}
}

type insufficientStake struct {
	Code        string `json:"code,omitempty"`
	PublicKey   string `json:"public_key,omitempty"`
	Owner       string `json:"owner,omitempty"`
	CoinSymbol  string `json:"coin_symbol,omitempty"`
	CoinId      string `json:"coin_id,omitempty"`
	StakeValue  string `json:"stake_value,omitempty"`
	NeededValue string `json:"needed_value,omitempty"`
}

func NewInsufficientStake(publicKey string, owner string, coinId string, coinSymbol string, stakeValue string, neededValue string) *insufficientStake {
	return &insufficientStake{Code: strconv.Itoa(int(InsufficientStake)), PublicKey: publicKey, Owner: owner, CoinId: coinId, CoinSymbol: coinSymbol, StakeValue: stakeValue, NeededValue: neededValue}
}

type tooLongNonce struct {
	Code          string `json:"code,omitempty"`
	NonceBytes    string `json:"nonce_bytes,omitempty"`
	MaxNonceBytes string `json:"max_nonce_bytes,omitempty"`
}

func NewTooLongNonce(nonceBytes string, maxNonceBytes string) *tooLongNonce {
	return &tooLongNonce{Code: strconv.Itoa(int(TooLongNonce)), NonceBytes: nonceBytes, MaxNonceBytes: maxNonceBytes}
}

type tooLargeOwnersList struct {
	Code           string `json:"code,omitempty"`
	CountOwners    string `json:"count_owners,omitempty"`
	MaxCountOwners string `json:"max_count_owners,omitempty"`
}

func NewTooLargeOwnersList(countOwners string, maxCountOwners string) *tooLargeOwnersList {
	return &tooLargeOwnersList{Code: strconv.Itoa(int(TooLargeOwnersList)), CountOwners: countOwners, MaxCountOwners: maxCountOwners}
}

type incorrectWeights struct {
	Code      string `json:"code,omitempty"`
	Address   string `json:"address,omitempty"`
	Weight    string `json:"weight,omitempty"`
	MaxWeight string `json:"max_weight,omitempty"`
}

func NewIncorrectWeights(address string, weight string, maxWeight string) *incorrectWeights {
	return &incorrectWeights{Code: strconv.Itoa(int(IncorrectWeights)), Address: address, Weight: weight, MaxWeight: maxWeight}
}

type incorrectTotalWeights struct {
	Code         string `json:"code,omitempty"`
	TotalWeights string `json:"total_weights,omitempty"`
	Threshold    string `json:"threshold,omitempty"`
}

func NewIncorrectTotalWeights(totalWeight, threshold string) *incorrectTotalWeights {
	return &incorrectTotalWeights{Code: strconv.Itoa(int(IncorrectTotalWeights)), Threshold: threshold, TotalWeights: totalWeight}
}

type differentCountAddressesAndWeights struct {
	Code           string `json:"code,omitempty"`
	CountAddresses string `json:"count_addresses,omitempty"`
	CountWeights   string `json:"count_weights,omitempty"`
}

func NewDifferentCountAddressesAndWeights(countAddresses string, countWeights string) *differentCountAddressesAndWeights {
	return &differentCountAddressesAndWeights{Code: strconv.Itoa(int(DifferentCountAddressesAndWeights)), CountAddresses: countAddresses, CountWeights: countWeights}
}

type minimumValueToBuyReached struct {
	Code              string `json:"code,omitempty"`
	MinimumValueToBuy string `json:"minimum_value_to_buy,omitempty"`
	WillGetValue      string `json:"will_get_value,omitempty"`
	CoinSymbol        string `json:"coin_symbol,omitempty"`
	CoinId            string `json:"coin_id,omitempty"`
}

func NewMinimumValueToBuyReached(minimumValueToBuy string, willGetValue string, coinSymbol string, coinId string) *minimumValueToBuyReached {
	return &minimumValueToBuyReached{Code: strconv.Itoa(int(MinimumValueToBuyReached)), MinimumValueToBuy: minimumValueToBuy, WillGetValue: willGetValue, CoinSymbol: coinSymbol, CoinId: coinId}
}

type maximumValueToSellReached struct {
	Code               string `json:"code,omitempty"`
	MaximumValueToSell string `json:"maximum_value_to_sell,omitempty"`
	NeededSpendValue   string `json:"needed_spend_value,omitempty"`
	CoinSymbol         string `json:"coin_symbol,omitempty"`
	CoinId             string `json:"coin_id,omitempty"`
}

func NewMaximumValueToSellReached(maximumValueToSell string, neededSpendValue string, coinSymbol string, coinId string) *maximumValueToSellReached {
	return &maximumValueToSellReached{Code: strconv.Itoa(int(MaximumValueToSellReached)), MaximumValueToSell: maximumValueToSell, NeededSpendValue: neededSpendValue, CoinSymbol: coinSymbol, CoinId: coinId}
}

type duplicatedAddresses struct {
	Code    string `json:"code,omitempty"`
	Address string `json:"address,omitempty"`
}

func NewDuplicatedAddresses(address string) *duplicatedAddresses {
	return &duplicatedAddresses{Code: strconv.Itoa(int(DuplicatedAddresses)), Address: address}
}

type checkInvalidLock struct {
	Code string `json:"code,omitempty"`
}

func NewCheckInvalidLock() *checkInvalidLock {
	return &checkInvalidLock{Code: strconv.Itoa(int(CheckInvalidLock))}
}

type crossConvert struct {
	Code         string `json:"code,omitempty"`
	CoinIdToSell string `json:"coin_id_to_sell,omitempty"`
	CoinToSell   string `json:"coin_to_sell,omitempty"`
	CoinIdToBuy  string `json:"coin_id_to_buy,omitempty"`
	CoinToBuy    string `json:"coin_to_buy,omitempty"`
}

func NewCrossConvert(coinIdToSell string, coinToSell string, coinIdToBuy string, coinToBuy string) *crossConvert {
	return &crossConvert{Code: strconv.Itoa(int(CrossConvert)), CoinIdToSell: coinIdToSell, CoinToSell: coinToSell, CoinIdToBuy: coinIdToBuy, CoinToBuy: coinToBuy}
}

type isNotOwnerOfCoin struct {
	Code       string  `json:"code,omitempty"`
	CoinSymbol string  `json:"coin_symbol,omitempty"`
	Owner      *string `json:"owner"`
}

func NewIsNotOwnerOfCoin(coinSymbol string, owner *string) *isNotOwnerOfCoin {
	var own *string
	if owner != nil {
		own = owner
	}
	return &isNotOwnerOfCoin{Code: strconv.Itoa(int(IsNotOwnerOfCoin)), CoinSymbol: coinSymbol, Owner: own}
}

type isNotOwnerOfCandidate struct {
	Code      string `json:"code,omitempty"`
	Sender    string `json:"sender,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Control   string `json:"control,omitempty"`
}

func NewIsNotOwnerOfCandidate(sender, pubKey string, owner, control string) *isNotOwnerOfCandidate {
	return &isNotOwnerOfCandidate{Code: strconv.Itoa(int(IsNotOwnerOfCandidate)), PublicKey: pubKey, Owner: owner, Control: control, Sender: sender}
}

type checkExpired struct {
	Code         string `json:"code,omitempty"`
	DueBlock     string `json:"due_block,omitempty"`
	CurrentBlock string `json:"current_block,omitempty"`
}

func MewCheckExpired(dueBlock string, currentBlock string) *checkExpired {
	return &checkExpired{Code: strconv.Itoa(int(CheckExpired)), DueBlock: dueBlock, CurrentBlock: currentBlock}
}

type checkUsed struct {
	Code string `json:"code,omitempty"`
}

func NewCheckUsed() *checkUsed {
	return &checkUsed{Code: strconv.Itoa(int(CheckUsed))}
}

type notEnoughMultisigVotes struct {
	Code        string `json:"code,omitempty"`
	NeededVotes string `json:"needed_votes,omitempty"`
	GotVotes    string `json:"got_votes,omitempty"`
}

func NewNotEnoughMultisigVotes(neededVotes, gotVotes string) *notEnoughMultisigVotes {
	return &notEnoughMultisigVotes{Code: strconv.Itoa(int(NotEnoughMultisigVotes)), NeededVotes: neededVotes, GotVotes: gotVotes}
}

type incorrectMultiSignature struct {
	Code string `json:"code,omitempty"`
}

func NewIncorrectMultiSignature() *incorrectMultiSignature {
	return &incorrectMultiSignature{Code: strconv.Itoa(int(IncorrectMultiSignature))}
}

type wrongCrr struct {
	Code   string `json:"code,omitempty"`
	MaxCrr string `json:"max_crr,omitempty"`
	MinCrr string `json:"min_crr,omitempty"`
	GotCrr string `json:"got_crr,omitempty"`
}

func NewWrongCrr(min string, max string, got string) *wrongCrr {
	return &wrongCrr{Code: strconv.Itoa(int(WrongCrr)), MinCrr: min, MaxCrr: max, GotCrr: got}
}

type stakeShouldBePositive struct {
	Code  string `json:"code,omitempty"`
	Stake string `json:"stake,omitempty"`
}

func NewStakeShouldBePositive(stake string) *stakeShouldBePositive {
	return &stakeShouldBePositive{Code: strconv.Itoa(int(StakeShouldBePositive)), Stake: stake}
}

type wrongHaltHeight struct {
	Code      string `json:"code,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
	Height    string `json:"height,omitempty"`
}

func NewWrongHaltHeight(height string, pubkey string) *wrongHaltHeight {
	return &wrongHaltHeight{Code: strconv.Itoa(int(WrongHaltHeight)), Height: height, PublicKey: pubkey}
}

type tooLowStake struct {
	Code       string `json:"code,omitempty"`
	Sender     string `json:"sender,omitempty"`
	PublicKey  string `json:"public_key,omitempty"`
	Value      string `json:"value,omitempty"`
	CoinSymbol string `json:"coin_symbol,omitempty"`
	CoinId     string `json:"coin_id,omitempty"`
}

func NewTooLowStake(sender string, pubKey string, value string, coinId string, coinSymbol string) *tooLowStake {
	return &tooLowStake{Code: strconv.Itoa(int(TooLowStake)), Sender: sender, PublicKey: pubKey, Value: value, CoinId: coinId, CoinSymbol: coinSymbol}
}

type wrongCommission struct {
	Code          string `json:"code,omitempty"`
	GotCommission string `json:"got_commission,omitempty"`
	MinCommission string `json:"min_commission,omitempty"`
	MaxCommission string `json:"max_commission,omitempty"`
}

func NewWrongCommission(got string, min string, max string) *wrongCommission {
	return &wrongCommission{Code: strconv.Itoa(int(WrongCommission)), MaxCommission: max, MinCommission: min, GotCommission: got}
}

type multisigNotExists struct {
	Code    string `json:"code,omitempty"`
	Address string `json:"address,omitempty"`
}

func NewMultisigNotExists(address string) *multisigNotExists {
	return &multisigNotExists{Code: strconv.Itoa(int(MultisigNotExists)), Address: address}
}

type multisigExists struct {
	Code    string `json:"code,omitempty"`
	Address string `json:"address,omitempty"`
}

func NewMultisigExists(address string) *multisigExists {
	return &multisigExists{Code: strconv.Itoa(int(MultisigExists)), Address: address}
}

type wrongCoinSupply struct {
	Code string `json:"code,omitempty"`

	MaxCoinSupply     string `json:"max_coin_supply,omitempty"`
	CurrentCoinSupply string `json:"current_coin_supply,omitempty"`

	MinInitialReserve     string `json:"min_initial_reserve,omitempty"`
	CurrentInitialReserve string `json:"current_initial_reserve,omitempty"`

	MinInitialAmount     string `json:"min_initial_amount,omitempty"`
	MaxInitialAmount     string `json:"max_initial_amount,omitempty"`
	CurrentInitialAmount string `json:"current_initial_amount,omitempty"`
}

func NewWrongCoinSupply(maxCoinSupply string, currentCoinSupply string, minInitialReserve string, currentInitialReserve string, minInitialAmount string, maxInitialAmount string, currentInitialAmount string) *wrongCoinSupply {
	return &wrongCoinSupply{Code: strconv.Itoa(int(WrongCoinSupply)), MaxCoinSupply: maxCoinSupply, CurrentCoinSupply: currentCoinSupply, MinInitialReserve: minInitialReserve, CurrentInitialReserve: currentInitialReserve, MinInitialAmount: minInitialAmount, MaxInitialAmount: maxInitialAmount, CurrentInitialAmount: currentInitialAmount}
}
