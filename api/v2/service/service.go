package service

import (
	"bytes"
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/golang/protobuf/jsonpb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/tendermint/go-amino"
	rpc "github.com/tendermint/tendermint/rpc/client"
	"google.golang.org/grpc/status"
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

func (s *Service) createError(statusErr *status.Status, details map[string]string) error {
	if len(details) == 0 {
		return statusErr.Err()
	}

	marshal, err := json.Marshal(details)
	if err != nil {
		s.client.Logger.Error(err.Error())
		return statusErr.Err()
	}

	var bb bytes.Buffer
	if _, err = bb.Write(marshal); err != nil {
		s.client.Logger.Error(err.Error())
		return statusErr.Err()
	}

	detailsMap := &_struct.Struct{Fields: make(map[string]*_struct.Value)}
	if err := (&jsonpb.Unmarshaler{}).Unmarshal(&bb, detailsMap); err != nil {
		s.client.Logger.Error(err.Error())
		return statusErr.Err()
	}

	withDetails, err := statusErr.WithDetails(detailsMap)
	if err != nil {
		s.client.Logger.Error(err.Error())
		return statusErr.Err()
	}

	return withDetails.Err()
}
