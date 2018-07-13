package types

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"math/big"
)

type UnbondData struct {
	PubKey []byte
	Coin   types.CoinSymbol
	Value  *big.Int
}

func (s UnbondData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string
		Coin   types.CoinSymbol
		Value  string
	}{
		PubKey: fmt.Sprintf("Mp%x", s.PubKey),
		Coin:   s.Coin,
		Value:  s.Value.String(),
	})
}

func (s UnbondData) String() string {
	return fmt.Sprintf("UNBOND pubkey:%s",
		hexutil.Encode(s.PubKey[:]))
}

func (s UnbondData) Gas() int64 {
	return commissions.UnbondTx
}
