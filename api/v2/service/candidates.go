package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Candidates returns list of candidates.
func (s *Service) Candidates(ctx context.Context, req *pb.CandidatesRequest) (*pb.CandidatesResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 {
		cState.Lock()
		cState.Candidates().LoadCandidates()
		if req.IncludeStakes {
			cState.Candidates().LoadStakes()
		}
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	candidates := cState.Candidates().GetCandidates()

	response := &pb.CandidatesResponse{}
	for _, candidate := range candidates {

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.CandidatesResponse), timeoutStatus.Err()
		}

		if req.Status != pb.CandidatesRequest_all && req.Status != pb.CandidatesRequest_CandidateStatus(candidate.Status) {
			continue
		}

		response.Candidates = append(response.Candidates, makeResponseCandidate(cState, candidate, req.IncludeStakes))
	}

	return response, nil
}
