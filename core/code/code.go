package code

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

	// coin creation
	CoinAlreadyExists uint32 = 201
	WrongCrr          uint32 = 202
	InvalidCoinSymbol uint32 = 203
	InvalidCoinName   uint32 = 204
	WrongCoinSupply   uint32 = 205

	// convert
	CrossConvert              uint32 = 301
	MaximumValueToSellReached uint32 = 302
	MinimumValueToBuylReached uint32 = 303

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

	// check
	CheckInvalidLock uint32 = 501
	CheckExpired     uint32 = 502
	CheckUsed        uint32 = 503
	TooHighGasPrice  uint32 = 504
	WrongGasCoin     uint32 = 505

	// multisig
	IncorrectWeights        uint32 = 601
	MultisigExists          uint32 = 602
	MultisigNotExists       uint32 = 603
	IncorrectMultiSignature uint32 = 604
	TooLargeOwnersList      uint32 = 605
)
