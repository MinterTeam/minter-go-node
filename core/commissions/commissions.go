package commissions

// all commissions are divided by 10^8
// actual commission is SendTx * 10^8 = 100 000 000 000 PIP = 0,0000001 BIP
const (
	SendTx                int64 = 1000000
	ConvertTx             int64 = 100000000
	CreateTx              int64 = 100000000
	DeclareCandidacyTx    int64 = 1000000000
	DelegateTx            int64 = 10000000
	UnboundTx             int64 = 10000000
	RedeemCheckTx         int64 = 1000000
	PayloadByte           int64 = 200000
	ToggleCandidateStatus int64 = 10000000
)
