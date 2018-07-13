package types

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
)

type RedeemCheckData struct {
	RawCheck []byte
	Proof    [65]byte
}

func (s RedeemCheckData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RawCheck string
		Proof    string
	}{
		RawCheck: fmt.Sprintf("Mc%x", s.RawCheck),
		Proof:    fmt.Sprintf("%x", s.Proof),
	})
}

func (s RedeemCheckData) String() string {
	return fmt.Sprintf("REDEEM CHECK proof: %x", s.Proof)
}

func (s RedeemCheckData) Gas() int64 {
	return commissions.SendTx
}
