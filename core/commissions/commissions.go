package commissions

// all commissions are divided by 10^8
// actual commission is SendTx * 10^8 = 100 000 000 000 PIP = 0,0000001 BIP
const (
	SendTx                int64 = 1000
	ConvertTx             int64 = 10000
	CreateTx              int64 = 100000
	DeclareCandidacyTx    int64 = 100000
	DelegateTx            int64 = 10000
	UnboundTx             int64 = 10000
	RedeemCheckTx         int64 = 1000
	PayloadByte           int64 = 500
	ToggleCandidateStatus int64 = 1000
)
