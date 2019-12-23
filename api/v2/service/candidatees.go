package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
)

func (s *Service) Candidates(_ context.Context, req *pb.CandidatesRequest) (*pb.CandidatesResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.CandidatesResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	candidates := cState.Candidates.GetCandidates()

	result := &pb.CandidatesResponse{
		Candidates: make([]*pb.CandidateResponse, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		result.Candidates = append(result.Candidates, makeResponseCandidate(cState, *candidate, req.IncludeStakes))
	}

	return result, nil
}
