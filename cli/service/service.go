package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/cli/cli_pb"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/tendermint/tendermint/evidence"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	typesTM "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"runtime"
	"time"
)

type Manager struct {
	blockchain *minter.Blockchain
	tmRPC      *rpc.Local
	tmNode     *tmNode.Node
	cfg        *config.Config
}

func NewManager(blockchain *minter.Blockchain, tmRPC *rpc.Local, tmNode *tmNode.Node, cfg *config.Config) pb.ManagerServiceServer {
	return &Manager{blockchain: blockchain, tmRPC: tmRPC, tmNode: tmNode, cfg: cfg}
}

func (m *Manager) Dashboard(_ *empty.Empty, stream pb.ManagerService_DashboardServer) error {
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
			resultStatus, err := m.tmRPC.Status()
			if err != nil {
				return status.Error(codes.Internal, err.Error())
			}
			netInfo, err := m.tmRPC.NetInfo()
			if err != nil {
				return status.Error(codes.Internal, err.Error())
			}
			pubKey := fmt.Sprintf("Mp%x", resultStatus.ValidatorInfo.PubKey.Bytes()[5:])

			state, err := m.blockchain.GetStateForHeight(0)
			if err != nil {
				return status.Error(codes.NotFound, err.Error())
			}

			var address types.TmAddress
			copy(address[:], resultStatus.ValidatorInfo.Address)
			validator := state.Validators().GetByTmAddress(address)
			validatorStatus := m.blockchain.GetValidatorStatus(address)

			var pbValidatorStatus pb.DashboardResponse_ValidatorStatus

			switch true {
			case validator != nil && validatorStatus == minter.ValidatorAbsent:
				pbValidatorStatus = pb.DashboardResponse_Validating
			case validator == nil && validatorStatus == minter.ValidatorAbsent:
				pbValidatorStatus = pb.DashboardResponse_Challenger
			case validator == nil && validatorStatus == minter.ValidatorPresent:
				pbValidatorStatus = pb.DashboardResponse_Offline
			default:
				pbValidatorStatus = pb.DashboardResponse_NotDeclared
			}

			var missedBlocks string
			var stake string
			if pbValidatorStatus == pb.DashboardResponse_Validating {
				missedBlocks = validator.AbsentTimes.String()
				stake = validator.GetTotalBipStake().String()
			}

			if err := stream.Send(&pb.DashboardResponse{
				LatestHeight:           info.Height,
				Timestamp:              protoTime,
				Duration:               info.Duration,
				MemoryUsage:            mem.Sys,
				ValidatorPubKey:        pubKey,
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

func (m *Manager) Status(context.Context, *empty.Empty) (*pb.StatusResponse, error) {
	result, err := m.tmRPC.Status()
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
		PublicKey:         fmt.Sprintf("Mp%x", result.ValidatorInfo.PubKey.Bytes()[5:]),
		NodeId:            string(result.NodeInfo.ID()),
	}

	return response, nil
}

func (m *Manager) NetInfo(context.Context, *empty.Empty) (*pb.NetInfoResponse, error) {
	resultNetInfo, err := m.tmRPC.NetInfo()
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
		peerHeight := peerHeight(m.tmNode.Switch(), peer.NodeInfo.ID())
		peers = append(peers, &pb.NetInfoResponse_Peer{
			LatestBlockHeight: peerHeight,
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

const countBatchBlocksDelete = 250

func (m *Manager) PruneBlocks(req *pb.PruneBlocksRequest, stream pb.ManagerService_PruneBlocksServer) error {
	current := m.blockchain.Height()
	if req.ToHeight >= int64(current) {
		return status.Errorf(codes.FailedPrecondition, "cannot delete latest saved version (%d)", current)
	}

	min := req.FromHeight - 1
	total := req.ToHeight - min

	from := req.FromHeight
	last := make(chan struct{})

	for i := req.FromHeight; i <= req.ToHeight; i++ {
		if i == req.ToHeight {
			close(last)
		}
		select {
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, stream.Context().Err().Error())
		case <-last:
			_ = m.blockchain.DeleteStateVersions(from, i)
			if err := stream.Send(&pb.PruneBlocksResponse{
				Total:   total,
				Current: i - min,
			}); err != nil {
				return err
			}
			return nil
		default:
			if i-from != countBatchBlocksDelete {
				continue
			}
			_ = m.blockchain.DeleteStateVersions(from, i)
			if err := stream.Send(&pb.PruneBlocksResponse{
				Total:   total,
				Current: i - min,
			}); err != nil {
				return err
			}
			from = i
		}
	}

	return nil
}

func (m *Manager) DealPeer(_ context.Context, req *pb.DealPeerRequest) (*empty.Empty, error) {
	res := new(empty.Empty)
	_, err := m.tmRPC.DialPeers([]string{req.Address}, req.Persistent)
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
