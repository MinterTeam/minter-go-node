package code

const (
	OK                       uint32 = 0
	DecodeError              uint32 = 1
	InsufficientFunds        uint32 = 2
	UnknownTransactionType   uint32 = 3
	WrongNonce               uint32 = 4
	CoinNotExists            uint32 = 5
	CoinAlreadyExists        uint32 = 6
	WrongCrr                 uint32 = 7
	CrossConvert             uint32 = 8
	CandidateExists          uint32 = 9
	WrongCommission          uint32 = 10
	CandidateNotFound        uint32 = 11
	CheckInvalidLock         uint32 = 12
	CheckExpired             uint32 = 13
	CheckUsed                uint32 = 14
	TooHighGasPrice          uint32 = 15
	StakeNotFound            uint32 = 16
	InsufficientStake        uint32 = 17
	IsNotOwnerOfCandidate    uint32 = 18
	CoinReserveNotSufficient uint32 = 19
	InvalidCoinSymbol        uint32 = 20
	TooLongPayload           uint32 = 21
)
