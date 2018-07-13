package types

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
)

type SetCandidateOnData struct {
	PubKey []byte
}

func (s SetCandidateOnData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string `json:"pubkey"`
	}{
		PubKey: fmt.Sprintf("Mp%x", s.PubKey),
	})
}

func (s SetCandidateOnData) String() string {
	return fmt.Sprintf("SET CANDIDATE ONLINE pubkey: %x",
		s.PubKey)
}

func (s SetCandidateOnData) Gas() int64 {
	return commissions.ToggleCandidateStatus
}

type SetCandidateOffData struct {
	PubKey []byte
}

func (s SetCandidateOffData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string `json:"pubkey"`
	}{
		PubKey: fmt.Sprintf("Mp%x", s.PubKey),
	})
}

func (s SetCandidateOffData) String() string {
	return fmt.Sprintf("SET CANDIDATE OFFLINE pubkey: %x",
		s.PubKey)
}

func (s SetCandidateOffData) Gas() int64 {
	return commissions.ToggleCandidateStatus
}
