package commissions

// all commissions are divided by 10^14
// actual commission is SendTx * 10^14 = 10 000 000 000 000 000 PIP = 0,01 BIP
const (
	SendTx                int64 = 10
	ConvertTx             int64 = 1000
	CreateTx              int64 = 1000
	DeclareCandidacyTx    int64 = 10000
	DelegateTx            int64 = 100
	UnbondTx              int64 = 100
	PayloadByte           int64 = 2
	ToggleCandidateStatus int64 = 100
)
