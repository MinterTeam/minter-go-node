package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/tendermint/tendermint/evidence"
	"github.com/tendermint/tendermint/p2p"
	typesTM "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

// NetInfo returns network info.
func (s *Service) NetInfo(ctx context.Context, _ *empty.Empty) (*pb.NetInfoResponse, error) {
	result, err := s.client.NetInfo()
	if err != nil {
		return new(pb.NetInfoResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	var peers []*pb.NetInfoResponse_Peer
	for _, peer := range result.Peers {

		var currentHeight *wrappers.UInt64Value
		peerHeight := peerHeight(s.tmNode.Switch(), peer.NodeInfo.ID())
		if peerHeight != 0 {
			currentHeight = &wrappers.UInt64Value{Value: uint64(peerHeight)}
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.NetInfoResponse), timeoutStatus.Err()
		}

		var channels []*pb.NetInfoResponse_Peer_ConnectionStatus_Channel
		for _, channel := range peer.ConnectionStatus.Channels {
			channels = append(channels, &pb.NetInfoResponse_Peer_ConnectionStatus_Channel{
				Id:                int64(channel.ID),
				SendQueueCapacity: int64(channel.SendQueueCapacity),
				SendQueueSize:     int64(channel.SendQueueSize),
				Priority:          int64(channel.Priority),
				RecentlySent:      channel.RecentlySent,
			})
		}

		peers = append(peers, &pb.NetInfoResponse_Peer{
			LatestBlockHeight: currentHeight,
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
				Channels:   peer.NodeInfo.Channels.String(),
				Moniker:    peer.NodeInfo.Moniker,
				Other: &pb.NodeInfo_Other{
					TxIndex:    peer.NodeInfo.Other.TxIndex,
					RpcAddress: peer.NodeInfo.Other.RPCAddress,
				},
			},
			IsOutbound: peer.IsOutbound,
			ConnectionStatus: &pb.NetInfoResponse_Peer_ConnectionStatus{
				Duration: uint64(peer.ConnectionStatus.Duration),
				SendMonitor: &pb.NetInfoResponse_Peer_ConnectionStatus_Monitor{
					Active:   peer.ConnectionStatus.SendMonitor.Active,
					Start:    peer.ConnectionStatus.SendMonitor.Start.Format(time.RFC3339Nano),
					Duration: peer.ConnectionStatus.SendMonitor.Duration.Nanoseconds(),
					Idle:     peer.ConnectionStatus.SendMonitor.Idle.Nanoseconds(),
					Bytes:    peer.ConnectionStatus.SendMonitor.Bytes,
					Samples:  peer.ConnectionStatus.SendMonitor.Samples,
					InstRate: peer.ConnectionStatus.SendMonitor.InstRate,
					CurRate:  peer.ConnectionStatus.SendMonitor.CurRate,
					AvgRate:  peer.ConnectionStatus.SendMonitor.AvgRate,
					PeakRate: peer.ConnectionStatus.SendMonitor.PeakRate,
					BytesRem: peer.ConnectionStatus.SendMonitor.BytesRem,
					TimeRem:  peer.ConnectionStatus.SendMonitor.TimeRem.Nanoseconds(),
					Progress: uint64(peer.ConnectionStatus.SendMonitor.Progress),
				},
				RecvMonitor: &pb.NetInfoResponse_Peer_ConnectionStatus_Monitor{
					Active:   peer.ConnectionStatus.RecvMonitor.Active,
					Start:    peer.ConnectionStatus.RecvMonitor.Start.Format(time.RFC3339Nano),
					Duration: peer.ConnectionStatus.RecvMonitor.Duration.Nanoseconds(),
					Idle:     peer.ConnectionStatus.RecvMonitor.Idle.Nanoseconds(),
					Bytes:    peer.ConnectionStatus.RecvMonitor.Bytes,
					Samples:  peer.ConnectionStatus.RecvMonitor.Samples,
					InstRate: peer.ConnectionStatus.RecvMonitor.InstRate,
					CurRate:  peer.ConnectionStatus.RecvMonitor.CurRate,
					AvgRate:  peer.ConnectionStatus.RecvMonitor.AvgRate,
					PeakRate: peer.ConnectionStatus.RecvMonitor.PeakRate,
					BytesRem: peer.ConnectionStatus.RecvMonitor.BytesRem,
					TimeRem:  peer.ConnectionStatus.RecvMonitor.TimeRem.Nanoseconds(),
					Progress: uint64(peer.ConnectionStatus.RecvMonitor.Progress),
				},
				Channels: channels,
			},
			RemoteIp: peer.RemoteIP,
		})
	}

	return &pb.NetInfoResponse{
		Listening:  result.Listening,
		Listeners:  result.Listeners,
		CountPeers: int64(result.NPeers),
		Peers:      peers,
	}, nil
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
