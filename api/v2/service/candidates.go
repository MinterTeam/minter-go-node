package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) Candidates(_ context.Context, req *pb.CandidatesRequest) (*pb.CandidatesResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.CandidatesResponse), status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 {
		cState.Lock()
		cState.Candidates.LoadCandidates()
		if req.IncludeStakes {
			cState.Candidates.LoadStakes()
		}
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	candidates := cState.Candidates.GetCandidates()

	result := &pb.CandidatesResponse{
		Candidates: make([]*pb.CandidateResponse, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		result.Candidates = append(result.Candidates, makeResponseCandidate(cState, *candidate, req.IncludeStakes))
	}

	return result, nil
}
