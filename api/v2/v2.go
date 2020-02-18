package v2

import (
	"context"
	gw "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"github.com/MinterTeam/minter-go-node/api/v2/service"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"net"
	"net/http"
)

func Run(srv *service.Service, addrGRPC, addrApi string) error {
	lis, err := net.Listen("tcp", addrGRPC)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	gw.RegisterApiServiceServer(grpcServer, srv)
	var group errgroup.Group

	group.Go(func() error {
		return grpcServer.Serve(lis)
	})

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
	opts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50000000))}

	group.Go(func() error {
		return gw.RegisterApiServiceHandlerFromEndpoint(ctx, mux, addrGRPC, opts)
	})
	group.Go(func() error {
		return http.ListenAndServe(addrApi, mux)
	})
	group.Go(func() error {
		return http.ListenAndServe(addrApi, wsproxy.WebsocketProxy(mux))
	})

	return group.Wait()
}
