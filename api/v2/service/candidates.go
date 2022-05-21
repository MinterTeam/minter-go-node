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

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	if req.Height != 0 {
		cState.Candidates().LoadCandidates()
		cState.Validators().LoadValidators()
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	candidates := cState.Candidates().GetCandidates()

	response := &pb.CandidatesResponse{}
	if req.Status == pb.CandidatesRequest_all || req.Status == pb.CandidatesRequest_deleted {
		for _, dc := range cState.Candidates().DeletedCandidates() {
			response.Deleted = append(response.Deleted, &pb.CandidatesResponse_Deleted{
				Id:        uint64(dc.ID),
				PublicKey: dc.PubKey.String(),
			})
		}
	}
	if req.Status == pb.CandidatesRequest_deleted {
		return response, nil
	}
	for _, candidate := range candidates {

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		isValidator := false
		if cState.Validators().GetByPublicKey(candidate.PubKey) != nil {
			isValidator = true
		}

		if req.Status != pb.CandidatesRequest_all {
			if req.Status == pb.CandidatesRequest_validator {
				if !isValidator {
					continue
				}
			} else if req.Status != pb.CandidatesRequest_CandidateStatus(candidate.Status) {
				continue
			}
		}

		if req.Height != 0 {
			cState.Candidates().LoadStakesOfCandidate(candidate.PubKey)
		}

		responseCandidate := s.makeResponseCandidate(ctx, cState, candidate, req.IncludeStakes, req.NotShowStakes)
		responseCandidate.Validator = isValidator

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		response.Candidates = append(response.Candidates, responseCandidate)
	}

	return response, nil
}
