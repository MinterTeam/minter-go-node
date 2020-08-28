package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func (s *Service) NetInfo(ctx context.Context, _ *empty.Empty) (*pb.NetInfoResponse, error) {
	result, err := s.client.NetInfo()
	if err != nil {
		return new(pb.NetInfoResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	var peers []*pb.NetInfoResponse_Peer
	for _, peer := range result.Peers {

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.NetInfoResponse), timeoutStatus.Err()
		}

		var channels []*pb.NetInfoResponse_Peer_ConnectionStatus_Channel
		for _, channel := range peer.ConnectionStatus.Channels {
			channels = append(channels, &pb.NetInfoResponse_Peer_ConnectionStatus_Channel{
				Id:                fmt.Sprintf("%d", channel.ID),
				SendQueueCapacity: fmt.Sprintf("%d", channel.SendQueueCapacity),
				SendQueueSize:     fmt.Sprintf("%d", channel.SendQueueSize),
				Priority:          fmt.Sprintf("%d", channel.Priority),
				RecentlySent:      fmt.Sprintf("%d", channel.RecentlySent),
			})
		}

		peers = append(peers, &pb.NetInfoResponse_Peer{
			NodeInfo: &pb.NodeInfo{
				ProtocolVersion: &pb.NodeInfo_ProtocolVersion{
					P2P:   fmt.Sprintf("%d", peer.NodeInfo.ProtocolVersion.P2P),
					Block: fmt.Sprintf("%d", peer.NodeInfo.ProtocolVersion.Block),
					App:   fmt.Sprintf("%d", peer.NodeInfo.ProtocolVersion.App),
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
				Duration: fmt.Sprintf("%d", peer.ConnectionStatus.Duration),
				SendMonitor: &pb.NetInfoResponse_Peer_ConnectionStatus_Monitor{
					Active:   false,
					Start:    peer.ConnectionStatus.SendMonitor.Start.Format(time.RFC3339Nano),
					Duration: fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.Duration.Nanoseconds()),
					Idle:     fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.Idle.Nanoseconds()),
					Bytes:    fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.Bytes),
					Samples:  fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.Samples),
					InstRate: fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.InstRate),
					CurRate:  fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.CurRate),
					AvgRate:  fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.AvgRate),
					PeakRate: fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.PeakRate),
					BytesRem: fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.BytesRem),
					TimeRem:  fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.TimeRem.Nanoseconds()),
					Progress: fmt.Sprintf("%d", peer.ConnectionStatus.SendMonitor.Progress),
				},
				RecvMonitor: nil,
				Channels:    channels,
			},
			RemoteIp: peer.RemoteIP,
		})
	}

	return &pb.NetInfoResponse{
		Listening:  result.Listening,
		Listeners:  result.Listeners,
		CountPeers: fmt.Sprintf("%d", result.NPeers),
		Peers:      peers,
	}, nil
}
