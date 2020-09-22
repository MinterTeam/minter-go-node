package service

import (
	"context"
	pb "github.com/MinterTeam/minter-go-node/cli/cli_pb"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
	"net"
	"os"
)

func StartCLIServer(socketPath string, manager pb.ManagerServiceServer, ctx context.Context) error {
	if err := os.RemoveAll(socketPath); err != nil {
		return err
	}

	lis, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return err
	}

	server := grpc.NewServer(
		grpc_middleware.WithStreamServerChain(
			grpc_recovery.StreamServerInterceptor(),
		),
		grpc_middleware.WithUnaryServerChain(
			grpc_recovery.UnaryServerInterceptor(),
		),
	)

	pb.RegisterManagerServiceServer(server, manager)

	kill := make(chan struct{})
	defer close(kill)
	go func() {
		select {
		case <-ctx.Done():
			server.GracefulStop()
		case <-kill:
		}
	}()

	if err := server.Serve(lis); err != nil {
		return err
	}

	return nil
}
