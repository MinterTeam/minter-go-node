package v2

import (
	"context"
	"github.com/MinterTeam/minter-go-node/api/v2/service"
	gw "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/gorilla/handlers"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
)

func Run(srv *service.Service, addrGRPC, addrApi string, traceLog bool) error {
	lis, err := net.Listen("tcp", addrGRPC)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc_middleware.WithStreamServerChain(
			grpc_prometheus.StreamServerInterceptor,
			grpc_recovery.StreamServerInterceptor(),
		),
		grpc_middleware.WithUnaryServerChain(
			grpc_prometheus.UnaryServerInterceptor,
			grpc_recovery.UnaryServerInterceptor(),
		),
	)
	gw.RegisterApiServiceServer(grpcServer, srv)
	grpc_prometheus.Register(grpcServer)

	var group errgroup.Group

	group.Go(func() error {
		return grpcServer.Serve(lis)
	})

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50000000)),
	}

	group.Go(func() error {
		return gw.RegisterApiServiceHandlerFromEndpoint(ctx, gwmux, addrGRPC, opts)
	})
	mux := http.NewServeMux()
	handler := wsproxy.WebsocketProxy(gwmux)

	if traceLog { //todo
		handler = handlers.CombinedLoggingHandler(os.Stdout, handler)
	}
	mux.Handle("/", handler)
	group.Go(func() error {
		return http.ListenAndServe(addrApi, mux)
	})

	return group.Wait()
}
