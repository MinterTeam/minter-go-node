package service

import (
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/tendermint/go-amino"
	rpc "github.com/tendermint/tendermint/rpc/client"
)

type Service struct {
	cdc        *amino.Codec
	blockchain *minter.Blockchain
	client     *rpc.Local
	minterCfg  *config.Config
	version    string
}

func NewService(cdc *amino.Codec, blockchain *minter.Blockchain, client *rpc.Local, minterCfg *config.Config, version string) *Service {
	return &Service{cdc: cdc, blockchain: blockchain, client: client, minterCfg: minterCfg, version: version}
}

func (s *Service) getStateForHeight(height int32) (*state.State, error) {
	if height > 0 {
		cState, err := s.blockchain.GetStateForHeight(uint64(height))
		if err != nil {
			return nil, err
		}
		return cState, nil
	}

	return s.blockchain.CurrentState(), nil
}
