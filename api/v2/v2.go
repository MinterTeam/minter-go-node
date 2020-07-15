package v2

import (
	"context"
	"github.com/MinterTeam/minter-go-node/api/v2/service"
	gw "github.com/MinterTeam/node-grpc-gateway/api_pb"
	_ "github.com/MinterTeam/node-grpc-gateway/statik"
	"github.com/gorilla/handlers"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rakyll/statik/fs"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"mime"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
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
			contextWithTimeoutInterceptor(srv.TimeoutDuration()),
		),
	)
	runtime.DefaultContextTimeout = 10 * time.Second

	gw.RegisterApiServiceServer(grpcServer, srv)
	grpc_prometheus.Register(grpcServer)

	var group errgroup.Group

	group.Go(func() error {
		return grpcServer.Serve(lis)
	})

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	gwmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}),
	)
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50000000)),
	}
	group.Go(func() error {
		return gw.RegisterApiServiceHandlerFromEndpoint(ctx, gwmux, addrGRPC, opts)
	})
	mux := http.NewServeMux()
	handler := wsproxy.WebsocketProxy(gwmux)

	if traceLog {
		handler = handlers.CombinedLoggingHandler(os.Stdout, handler)
	}
	mux.Handle("/", handler)
	if err := serveOpenAPI(mux); err != nil {
		//ignore
	}

	group.Go(func() error {
		return http.ListenAndServe(addrApi, allowCORS(mux))
	})

	return group.Wait()
}

func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func preflightHandler(w http.ResponseWriter, _ *http.Request) {
	headers := []string{"Content-Type", "Accept"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
	return
}

func serveOpenAPI(mux *http.ServeMux) error {
	_ = mime.AddExtensionType(".svg", "image/svg+xml")

	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	// Expose files in static on <host>/openapi-ui
	fileServer := http.FileServer(statikFS)
	prefix := "/openapi-ui/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
	return nil
}

func contextWithTimeoutInterceptor(timeout time.Duration) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		withTimeout, _ := context.WithTimeout(ctx, timeout)
		return handler(withTimeout, req)
	}
}
