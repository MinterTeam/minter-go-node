package commissions

// all commissions are divided by 10^15
// actual commission is SendTx * 10^15 = 10 000 000 000 000 000 PIP = 0,01 BIP
const (
	SendTx                 int64 = 10
	CreateMultisig         int64 = 100
	ConvertTx              int64 = 100
	DeclareCandidacyTx     int64 = 10000
	DelegateTx             int64 = 200
	UnbondTx               int64 = 200
	PayloadByte            int64 = 2
	ToggleCandidateStatus  int64 = 100
	EditCandidate          int64 = 10000
	EditCandidatePublicKey int64 = 100000000
	MultisendDelta         int64 = 5
	RedeemCheckTx                = SendTx * 3
	SetHaltBlock           int64 = 1000
	RecreateCoin           int64 = 10000000
	EditOwner              int64 = 10000000
	EditMultisigData       int64 = 1000
	PriceVoteData          int64 = 10
)
