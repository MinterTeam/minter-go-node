package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cli/pb"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	rpc "github.com/tendermint/tendermint/rpc/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"runtime"
	"time"
)

type Manager struct {
	blockchain *minter.Blockchain
	tmRPC      *rpc.Local
	cfg        *config.Config
}

func NewManager(blockchain *minter.Blockchain, tmRPC *rpc.Local, cfg *config.Config) pb.ManagerServiceServer {
	return &Manager{blockchain: blockchain, tmRPC: tmRPC, cfg: cfg}
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
			maxPeersHeight := m.blockchain.MaxPeerHeight()
			maxPeersHeight = maxPeersHeight - 1
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
	resultStatus, err := m.tmRPC.Status()
	if err != nil {
		return new(pb.StatusResponse), status.Error(codes.Internal, err.Error())
	}

	response := &pb.StatusResponse{
		Version:           version.Version,
		LatestBlockHash:   fmt.Sprintf("%X", resultStatus.SyncInfo.LatestBlockHash),
		LatestAppHash:     fmt.Sprintf("%X", resultStatus.SyncInfo.LatestAppHash),
		KeepLastStates:    int64(m.blockchain.CurrentState().Tree().KeepLastHeight()),
		LatestBlockHeight: resultStatus.SyncInfo.LatestBlockHeight,
		LatestBlockTime:   resultStatus.SyncInfo.LatestBlockTime.Format(time.RFC3339),
		TmStatus: &pb.StatusResponse_TmStatus{
			NodeInfo: &pb.NodeInfo{
				ProtocolVersion: &pb.NodeInfo_ProtocolVersion{
					P2P:   uint64(resultStatus.NodeInfo.ProtocolVersion.P2P),
					Block: uint64(resultStatus.NodeInfo.ProtocolVersion.Block),
					App:   uint64(resultStatus.NodeInfo.ProtocolVersion.App),
				},
				Id:         fmt.Sprintf("%X", resultStatus.NodeInfo.ID()),
				ListenAddr: resultStatus.NodeInfo.ListenAddr,
				Network:    resultStatus.NodeInfo.Network,
				Version:    resultStatus.NodeInfo.Version,
				Channels:   fmt.Sprintf("%X", resultStatus.NodeInfo.Channels),
				Moniker:    resultStatus.NodeInfo.Moniker,
				Other: &pb.NodeInfo_Other{
					TxIndex:    resultStatus.NodeInfo.Other.TxIndex,
					RpcAddress: resultStatus.NodeInfo.Other.RPCAddress,
				},
			},
			SyncInfo: &pb.StatusResponse_TmStatus_SyncInfo{
				LatestBlockHash:   fmt.Sprintf("%X", resultStatus.SyncInfo.LatestBlockHash),
				LatestAppHash:     fmt.Sprintf("%X", resultStatus.SyncInfo.LatestAppHash),
				LatestBlockHeight: resultStatus.SyncInfo.LatestBlockHeight,
				LatestBlockTime:   resultStatus.SyncInfo.LatestBlockTime.Format(time.RFC3339),
				CatchingUp:        resultStatus.SyncInfo.CatchingUp,
			},
			ValidatorInfo: &pb.StatusResponse_TmStatus_ValidatorInfo{
				Address: fmt.Sprintf("%X", resultStatus.ValidatorInfo.Address),
				PubKey: &pb.StatusResponse_TmStatus_ValidatorInfo_PubKey{
					Type:  "tendermint/PubKeyEd25519",
					Value: fmt.Sprintf("%X", resultStatus.ValidatorInfo.PubKey.Bytes()),
				},
				VotingPower: resultStatus.ValidatorInfo.VotingPower,
			},
		},
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
				ID:                int32(channel.ID),
				SendQueueCapacity: int64(channel.SendQueueCapacity),
				SendQueueSize:     int64(channel.SendQueueSize),
				Priority:          int64(channel.Priority),
				RecentlySent:      channel.RecentlySent,
			})
		}
		peers = append(peers, &pb.NetInfoResponse_Peer{
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
		Listening: resultNetInfo.Listening,
		Listeners: resultNetInfo.Listeners,
		NPeers:    int64(resultNetInfo.NPeers),
		Peers:     peers,
	}

	return response, nil
}

func (m *Manager) PruneBlocks(ctx context.Context, req *pb.PruneBlocksRequest) (*empty.Empty, error) {
	return new(empty.Empty), status.Error(codes.Unimplemented, "todo")
}

func (m *Manager) DealPeer(ctx context.Context, req *pb.DealPeerRequest) (*empty.Empty, error) {
	res := new(empty.Empty)
	_, err := m.tmRPC.DialPeers([]string{req.Address}, req.Persistent)
	if err != nil {
		return res, status.Error(codes.FailedPrecondition, err.Error())
	}
	return res, nil
}
