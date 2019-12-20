package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/golang/protobuf/ptypes/empty"
	"time"
)

func (s *Service) NetInfo(context.Context, *empty.Empty) (*pb.NetInfoResponse, error) {
	result, err := s.client.NetInfo()
	if err != nil {
		return &pb.NetInfoResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	var peers []*pb.NetInfoResponse_Result_Peer
	for _, peer := range result.Peers {
		var channels []*pb.NetInfoResponse_Result_Peer_ConnectionStatus_Channel
		for _, channel := range peer.ConnectionStatus.Channels {
			channels = append(channels, &pb.NetInfoResponse_Result_Peer_ConnectionStatus_Channel{
				ID:                fmt.Sprintf("%d", channel.ID),
				SendQueueCapacity: fmt.Sprintf("%d", channel.SendQueueCapacity),
				SendQueueSize:     fmt.Sprintf("%d", channel.SendQueueSize),
				Priority:          fmt.Sprintf("%d", channel.Priority),
				RecentlySent:      fmt.Sprintf("%d", channel.RecentlySent),
			})
		}

		peers = append(peers, &pb.NetInfoResponse_Result_Peer{
			NodeInfo: &pb.NodeInfo{
				ProtocolVersion: &pb.NodeInfo_ProtocolVersion{
					P2P:   fmt.Sprintf("%d", peer.NodeInfo.ProtocolVersion.P2P),
					Block: fmt.Sprintf("%d", peer.NodeInfo.ProtocolVersion.Block),
					App:   fmt.Sprintf("%d", peer.NodeInfo.ProtocolVersion.App),
				},
				Id:         string(peer.NodeInfo.ID_),
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
			ConnectionStatus: &pb.NetInfoResponse_Result_Peer_ConnectionStatus{
				Duration: fmt.Sprintf("%d", peer.ConnectionStatus.Duration),
				SendMonitor: &pb.NetInfoResponse_Result_Peer_ConnectionStatus_Monitor{
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
		Jsonrpc: "2.0",
		Id:      "",
		Result: &pb.NetInfoResponse_Result{
			Listening: result.Listening,
			Listeners: result.Listeners,
			NPeers:    fmt.Sprintf("%d", result.NPeers),
			Peers:     peers,
		},
	}, nil
}
