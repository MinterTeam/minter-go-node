package types

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"math/big"
)

type DeclareCandidacyData struct {
	Address    types.Address
	PubKey     []byte
	Commission uint
	Coin       types.CoinSymbol
	Stake      *big.Int
}

func (s DeclareCandidacyData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Address    types.Address
		PubKey     string
		Commission uint
		Coin       types.CoinSymbol
		Stake      string
	}{
		Address:    s.Address,
		PubKey:     fmt.Sprintf("Mp%x", s.PubKey),
		Commission: s.Commission,
		Coin:       s.Coin,
		Stake:      s.Stake.String(),
	})
}

func (s DeclareCandidacyData) String() string {
	return fmt.Sprintf("DECLARE CANDIDACY address:%s pubkey:%s commission: %d ",
		s.Address.String(), hexutil.Encode(s.PubKey[:]), s.Commission)
}

func (s DeclareCandidacyData) Gas() int64 {
	return commissions.DeclareCandidacyTx
}
