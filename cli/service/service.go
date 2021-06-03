package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/cli/cli_pb"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/coreV2/minter"
	"github.com/MinterTeam/minter-go-node/coreV2/state/candidates"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/evidence"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	typesTM "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"math/big"
	"runtime"
	"time"
)

type managerServer struct {
	blockchain *minter.Blockchain
	tmRPC      *rpc.Local
	tmNode     *tmNode.Node
	cfg        *config.Config
	pb.UnimplementedManagerServiceServer
}

// NewManager return backend for cli
func NewManager(blockchain *minter.Blockchain, tmRPC *rpc.Local, tmNode *tmNode.Node, cfg *config.Config) pb.ManagerServiceServer {
	return &managerServer{blockchain: blockchain, tmRPC: tmRPC, tmNode: tmNode, cfg: cfg}
}

func (m *managerServer) Dashboard(_ *empty.Empty, stream pb.ManagerService_DashboardServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-time.After(time.Second):
			statisticData := m.blockchain.StatisticData()
			if statisticData == nil {
				return status.Error(codes.Unavailable, "Dashboard is not available, please enable prometheus in configuration")
			}
			info := statisticData.GetLastBlockInfo()
			averageTimeBlock := statisticData.GetAverageBlockProcessingTime()
			timePerBlock := statisticData.GetTimePerBlock()
			maxPeersHeight := maxPeerHeight(m.tmNode.Switch())
			if maxPeersHeight == 0 {
				maxPeersHeight = info.Height
			} else {
				maxPeersHeight = maxPeersHeight - 1
			}
			protoTime, _ := ptypes.TimestampProto(info.HeaderTimestamp)
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			resultStatus, err := m.tmRPC.Status(context.Background())
			if err != nil {
				return status.Error(codes.Internal, err.Error())
			}
			netInfo, err := m.tmRPC.NetInfo(context.Background())
			if err != nil {
				return status.Error(codes.Internal, err.Error())
			}

			var missedBlocks string
			var stake string
			pubkey := types.BytesToPubkey(resultStatus.ValidatorInfo.PubKey.Bytes()[:])
			var pbValidatorStatus pb.DashboardResponse_ValidatorStatus
			cState := m.blockchain.CurrentState()
			candidate := cState.Candidates().GetCandidate(pubkey)
			if candidate == nil {
				pbValidatorStatus = pb.DashboardResponse_NotDeclared
			} else {
				stake = big.NewInt(0).Div(candidate.GetTotalBipStake(), big.NewInt(1e18)).String() + " BIP"
				if candidate.Status == candidates.CandidateStatusOffline {
					pbValidatorStatus = pb.DashboardResponse_Offline
				} else {
					pbValidatorStatus = pb.DashboardResponse_Challenger
					validator := cState.Validators().GetByPublicKey(pubkey)
					if validator != nil {
						missedBlocks = validator.AbsentTimes.String()
						var address types.TmAddress
						copy(address[:], ed25519.PubKey(pubkey[:]).Address().Bytes())
						if m.blockchain.GetValidatorStatus(address) == minter.ValidatorPresent {
							pbValidatorStatus = pb.DashboardResponse_Validating
						}
					}
				}
			}

			if err := stream.Send(&pb.DashboardResponse{
				InitialHeight:          cState.InitialHeight(),
				LatestHeight:           info.Height,
				Timestamp:              protoTime,
				Duration:               info.Duration,
				MemoryUsage:            mem.Sys,
				ValidatorPubKey:        pubkey.String(),
				MaxPeerHeight:          maxPeersHeight,
				PeersCount:             int32(netInfo.NPeers),
				AvgBlockProcessingTime: averageTimeBlock,
				TimePerBlock:           timePerBlock,
				MissedBlocks:           missedBlocks,
				Stake:                  stake,
				VotingPower:            resultStatus.ValidatorInfo.VotingPower,
				ValidatorStatus:        pbValidatorStatus,
			}); err != nil {
				return err
			}
		}
	}
}

func (m *managerServer) Status(context.Context, *empty.Empty) (*pb.StatusResponse, error) {
	result, err := m.tmRPC.Status(context.Background())
	if err != nil {
		return new(pb.StatusResponse), status.Error(codes.Internal, err.Error())
	}

	response := &pb.StatusResponse{
		Version:           version.Version,
		LatestBlockHash:   fmt.Sprintf("%X", result.SyncInfo.LatestBlockHash),
		LatestAppHash:     fmt.Sprintf("%X", result.SyncInfo.LatestAppHash),
		LatestBlockHeight: fmt.Sprintf("%d", result.SyncInfo.LatestBlockHeight),
		LatestBlockTime:   result.SyncInfo.LatestBlockTime.Format(time.RFC3339Nano),
		KeepLastStates:    fmt.Sprintf("%d", m.cfg.BaseConfig.KeepLastStates),
		CatchingUp:        result.SyncInfo.CatchingUp,
		PublicKey:         fmt.Sprintf("Mp%x", result.ValidatorInfo.PubKey.Bytes()[:]),
		NodeId:            string(result.NodeInfo.ID()),
	}

	return response, nil
}

func (m *managerServer) NetInfo(context.Context, *empty.Empty) (*pb.NetInfoResponse, error) {
	resultNetInfo, err := m.tmRPC.NetInfo(context.Background())
	if err != nil {
		return new(pb.NetInfoResponse), status.Error(codes.Internal, err.Error())
	}

	peers := make([]*pb.NetInfoResponse_Peer, 0, len(resultNetInfo.Peers))
	for _, peer := range resultNetInfo.Peers {
		channels := make([]*pb.NetInfoResponse_Peer_ConnectionStatus_Channel, 0, len(peer.ConnectionStatus.Channels))
		for _, channel := range peer.ConnectionStatus.Channels {
			channels = append(channels, &pb.NetInfoResponse_Peer_ConnectionStatus_Channel{
				Id:                int32(channel.ID),
				SendQueueCapacity: int64(channel.SendQueueCapacity),
				SendQueueSize:     int64(channel.SendQueueSize),
				Priority:          int64(channel.Priority),
				RecentlySent:      channel.RecentlySent,
			})
		}
		var peerHeightValue *wrapperspb.Int64Value
		peerHeight := peerHeight(m.tmNode.Switch(), peer.NodeInfo.ID())
		if peerHeight != 0 {
			peerHeightValue = wrapperspb.Int64(peerHeight)
		}
		peers = append(peers, &pb.NetInfoResponse_Peer{
			LatestBlockHeight: peerHeightValue,
			NodeInfo: &pb.NodeInfo{
				ProtocolVersion: &pb.NodeInfo_ProtocolVersion{
					P2P:   uint64(peer.NodeInfo.ProtocolVersion.P2P),
					Block: uint64(peer.NodeInfo.ProtocolVersion.Block),
					App:   uint64(peer.NodeInfo.ProtocolVersion.App),
				},
				Id:         string(peer.NodeInfo.ID()),
				ListenAddr: peer.NodeInfo.ListenAddr,
				Network:    peer.NodeInfo.Network,
				Version:    peer.NodeInfo.Version,
				Channels:   string(peer.NodeInfo.Channels),
				Moniker:    peer.NodeInfo.Moniker,
				Other: &pb.NodeInfo_Other{
					TxIndex:    peer.NodeInfo.Other.TxIndex,
					RpcAddress: peer.NodeInfo.Other.RPCAddress,
				},
			},
			IsOutbound: peer.IsOutbound,
			ConnectionStatus: &pb.NetInfoResponse_Peer_ConnectionStatus{
				Duration: int64(peer.ConnectionStatus.Duration),
				SendMonitor: &pb.NetInfoResponse_Peer_ConnectionStatus_Monitor{
					Active:   peer.ConnectionStatus.SendMonitor.Active,
					Start:    peer.ConnectionStatus.SendMonitor.Start.Format(time.RFC3339),
					Duration: int64(peer.ConnectionStatus.SendMonitor.Duration),
					Idle:     int64(peer.ConnectionStatus.SendMonitor.Idle),
					Bytes:    peer.ConnectionStatus.SendMonitor.Bytes,
					Samples:  peer.ConnectionStatus.SendMonitor.Samples,
					InstRate: peer.ConnectionStatus.SendMonitor.InstRate,
					CurRate:  peer.ConnectionStatus.SendMonitor.CurRate,
					AvgRate:  peer.ConnectionStatus.SendMonitor.AvgRate,
					PeakRate: peer.ConnectionStatus.SendMonitor.PeakRate,
					BytesRem: peer.ConnectionStatus.SendMonitor.BytesRem,
					TimeRem:  int64(peer.ConnectionStatus.SendMonitor.TimeRem),
					Progress: uint32(peer.ConnectionStatus.SendMonitor.Progress),
				},
				RecvMonitor: &pb.NetInfoResponse_Peer_ConnectionStatus_Monitor{
					Active:   peer.ConnectionStatus.RecvMonitor.Active,
					Start:    peer.ConnectionStatus.RecvMonitor.Start.Format(time.RFC3339),
					Duration: int64(peer.ConnectionStatus.RecvMonitor.Duration),
					Idle:     int64(peer.ConnectionStatus.RecvMonitor.Idle),
					Bytes:    peer.ConnectionStatus.RecvMonitor.Bytes,
					Samples:  peer.ConnectionStatus.RecvMonitor.Samples,
					InstRate: peer.ConnectionStatus.RecvMonitor.InstRate,
					CurRate:  peer.ConnectionStatus.RecvMonitor.CurRate,
					AvgRate:  peer.ConnectionStatus.RecvMonitor.AvgRate,
					PeakRate: peer.ConnectionStatus.RecvMonitor.PeakRate,
					BytesRem: peer.ConnectionStatus.RecvMonitor.BytesRem,
					TimeRem:  int64(peer.ConnectionStatus.RecvMonitor.TimeRem),
					Progress: uint32(peer.ConnectionStatus.RecvMonitor.Progress),
				},
				Channels: channels,
			},
			RemoteIp: peer.RemoteIP,
		})
	}

	response := &pb.NetInfoResponse{
		Listening:  resultNetInfo.Listening,
		Listeners:  resultNetInfo.Listeners,
		CountPeers: int64(resultNetInfo.NPeers),
		Peers:      peers,
	}

	return response, nil
}

func (m *managerServer) AvailableVersions(context.Context, *empty.Empty) (*pb.AvailableVersionsResponse, error) {
	versions := m.blockchain.AvailableVersions()
	intervals := map[int]int{}
	var fromVersion int64
	for i := 0; i < len(versions); i++ {
		if versions[i]-versions[fromVersion]-1 != intervals[versions[fromVersion]] {
			fromVersion = int64(i)
		}
		if _, ok := intervals[versions[fromVersion]]; !ok {
			intervals[versions[fromVersion]] = 0
			continue
		}
		intervals[versions[fromVersion]]++
	}
	heights := make([]string, 0, len(intervals))
	for fromVersion, sortedBatchSize := range intervals {
		if sortedBatchSize == 0 {
			heights = append(heights, fmt.Sprintf("%d", fromVersion))
		} else {
			heights = append(heights, fmt.Sprintf("%d-%d", fromVersion, fromVersion+sortedBatchSize))
		}
	}
	return &pb.AvailableVersionsResponse{Heights: heights}, nil
}

func (m *managerServer) PruneBlocks(req *pb.PruneBlocksRequest, stream pb.ManagerService_PruneBlocksServer) error {
	total := req.ToHeight - req.FromHeight

	last := make(chan struct{})
	from := req.FromHeight
	for i := req.FromHeight + req.Batch; i < req.ToHeight+req.Batch; i += req.Batch {
		if i >= req.ToHeight {
			close(last)
		}
		select {
		case <-stream.Context().Done():
			return status.FromContextError(stream.Context().Err()).Err()
		case <-last:
			err := m.blockchain.DeleteStateVersions(from, req.ToHeight)
			if err != nil {
				return status.Error(codes.Aborted, err.Error())
			}
			if err := stream.Send(&pb.PruneBlocksResponse{
				Total:   total,
				Current: total,
			}); err != nil {
				return err
			}
			return nil
		default:
			err := m.blockchain.DeleteStateVersions(from, i)
			if err != nil {
				return status.Error(codes.Aborted, err.Error())
			}
			if err := stream.Send(&pb.PruneBlocksResponse{
				Total:   total,
				Current: i - req.FromHeight,
			}); err != nil {
				return err
			}
			from = i
			runtime.Gosched()
		}
	}

	return nil
}

func (m *managerServer) DealPeer(_ context.Context, req *pb.DealPeerRequest) (*empty.Empty, error) {
	res := new(empty.Empty)
	_, err := m.tmRPC.DialPeers(context.Background(), []string{req.Address}, req.Persistent, req.Unconditional, req.Private)
	if err != nil {
		return res, status.Error(codes.FailedPrecondition, err.Error())
	}
	return res, nil
}

func maxPeerHeight(sw *p2p.Switch) int64 {
	var max int64
	for _, peer := range sw.Peers().List() {
		peerState, ok := peer.Get(typesTM.PeerStateKey).(evidence.PeerState)
		if !ok {
			continue
		}
		height := peerState.GetHeight()
		if height > max {
			max = height
		}
	}
	return max
}

func peerHeight(sw *p2p.Switch, id p2p.ID) int64 {
	peerTM := sw.Peers().Get(id)
	if peerTM == nil {
		return 0
	}
	ps := peerTM.Get(typesTM.PeerStateKey)
	if ps == nil {
		return 0
	}
	peerState, ok := ps.(evidence.PeerState)
	if !ok {
		return 0
	}
	return peerState.GetHeight()
}
