package service

import (
	"context"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) Candidates(_ context.Context, req *pb.CandidatesRequest) (*pb.CandidatesResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.CandidatesResponse), status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 {
		cState.Candidates.LoadCandidates()
		if req.IncludeStakes {
			cState.Candidates.LoadStakes()
		}
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
