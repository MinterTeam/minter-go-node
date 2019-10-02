package old

import (
	"io"

	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
)

const (
	CandidateStatusOffline = 0x01
	CandidateStatusOnline  = 0x02
)

// stateCandidate represents a candidate which is being modified.
type stateCandidates struct {
	data Candidates
	db   *StateDB

	onDirty func() // Callback method to mark a state object newly dirty
}

type Candidates []Candidate

type Stake struct {
	Owner    types.Address
	Coin     types.CoinSymbol
	Value    *big.Int
	BipValue *big.Int
}

func (s *Stake) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Owner    types.Address    `json:"owner"`
		Coin     types.CoinSymbol `json:"coin"`
		Value    string           `json:"value"`
		BipValue string           `json:"bip_value"`
	}{
		Owner:    s.Owner,
		Coin:     s.Coin,
		Value:    s.Value.String(),
		BipValue: s.BipValue.String(),
	})
}

func (s *Stake) CalcSimulatedBipValue(context *StateDB) *big.Int {
	if s.Coin.IsBaseCoin() {
		return big.NewInt(0).Set(s.Value)
	}

	if s.Value.Cmp(types.Big0) == 0 {
		return big.NewInt(0)
	}

	totalStaked := big.NewInt(0)
	totalStaked.Add(totalStaked, s.Value)

	candidates := context.getStateCandidates()

	for _, candidate := range candidates.data {
		for _, stake := range candidate.Stakes {
			if stake.Coin == s.Coin {
				totalStaked.Add(totalStaked, stake.Value)
			}
		}
	}

	coin := context.getStateCoin(s.Coin)
	bipValue := formula.CalculateSaleReturn(coin.Volume(), coin.ReserveBalance(), coin.data.Crr, totalStaked)

	value := big.NewInt(0).Set(bipValue)
	value.Mul(value, s.Value)
	value.Div(value, totalStaked)

	return value
}

func (s *Stake) CalcBipValue(context *StateDB) *big.Int {
	if s.Coin.IsBaseCoin() {
		return big.NewInt(0).Set(s.Value)
	}

	if s.Value.Cmp(types.Big0) == 0 {
		return big.NewInt(0)
	}

	if _, has := context.stakeCache[s.Coin]; !has {
		totalStaked := big.NewInt(0)
		candidates := context.getStateCandidates()

		for _, candidate := range candidates.data {
			for _, stake := range candidate.Stakes {
				if stake.Coin == s.Coin {
					totalStaked.Add(totalStaked, stake.Value)
				}
			}
		}

		coin := context.getStateCoin(s.Coin)
		context.stakeCache[s.Coin] = StakeCache{
			TotalValue: totalStaked,
			BipValue:   formula.CalculateSaleReturn(coin.Volume(), coin.ReserveBalance(), coin.data.Crr, totalStaked),
		}
	}

	data := context.stakeCache[s.Coin]

	if data.TotalValue.Cmp(types.Big0) == 0 {
		return big.NewInt(0)
	}

	value := big.NewInt(0).Set(data.BipValue)
	value.Mul(value, s.Value)
	value.Div(value, data.TotalValue)

	return value
}

type Candidate struct {
	RewardAddress  types.Address
	OwnerAddress   types.Address
	TotalBipStake  *big.Int
	PubKey         types.Pubkey
	Commission     uint
	Stakes         []Stake
	CreatedAtBlock uint
	Status         byte

	tmAddress *[20]byte
}

func (candidate Candidate) GetStakeOfAddress(addr types.Address, coin types.CoinSymbol) *Stake {
	for i, stake := range candidate.Stakes {
		if bytes.Equal(stake.Coin.Bytes(), coin.Bytes()) && bytes.Equal(stake.Owner.Bytes(), addr.Bytes()) {
			return &(candidate.Stakes[i])
		}
	}

	return nil
}

func (candidate Candidate) String() string {
	return fmt.Sprintf("Candidate")
}

func (candidate Candidate) GetAddress() [20]byte {
	if candidate.tmAddress != nil {
		return *candidate.tmAddress
	}

	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], candidate.PubKey)

	var address [20]byte
	copy(address[:], pubkey.Address().Bytes())

	candidate.tmAddress = &address

	return address
}

// newCandidate creates a state object.
func newCandidate(db *StateDB, data Candidates, onDirty func()) *stateCandidates {
	candidate := &stateCandidates{
		db:      db,
		data:    data,
		onDirty: onDirty,
	}

	candidate.onDirty()

	return candidate
}

// EncodeRLP implements rlp.Encoder.
func (c *stateCandidates) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (c *stateCandidates) GetData() Candidates {
	return c.data
}
