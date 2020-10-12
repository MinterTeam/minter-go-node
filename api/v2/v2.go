package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/service"
	gw "github.com/MinterTeam/node-grpc-gateway/api_pb"
	_ "github.com/MinterTeam/node-grpc-gateway/statik"
	kit_log "github.com/go-kit/kit/log"
	"github.com/gorilla/handlers"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/kit"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rakyll/statik/fs"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	_struct "google.golang.org/protobuf/types/known/structpb"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Run initialises gRPC and API v2 interfaces
func Run(srv *service.Service, addrGRPC, addrApi string, logger log.Logger) error {
	lis, err := net.Listen("tcp", addrGRPC)
	if err != nil {
		return err
	}

	kitLogger := &kitLogger{logger}

	loggerOpts := []kit.Option{
		kit.WithLevels(func(code codes.Code, logger kit_log.Logger) kit_log.Logger { return logger }),
	}
	grpcServer := grpc.NewServer(
		grpc_middleware.WithStreamServerChain(
			grpc_prometheus.StreamServerInterceptor,
			grpc_recovery.StreamServerInterceptor(),
			grpc_ctxtags.StreamServerInterceptor(requestExtractorFields()),
			kit.StreamServerInterceptor(kitLogger, loggerOpts...),
		),
		grpc_middleware.WithUnaryServerChain(
			grpc_prometheus.UnaryServerInterceptor,
			grpc_recovery.UnaryServerInterceptor(),
			grpc_ctxtags.UnaryServerInterceptor(requestExtractorFields()),
			kit.UnaryServerInterceptor(kitLogger, loggerOpts...),
			unaryTimeoutInterceptor(srv.TimeoutDuration()),
		),
	)
	runtime.GlobalHTTPErrorHandler = httpError
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
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(200000000)),
	}
	err = gw.RegisterApiServiceHandlerFromEndpoint(ctx, gwmux, addrGRPC, opts)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	openapi := "/v2/openapi-ui/"
	_ = serveOpenAPI(openapi, mux)
	mux.HandleFunc("/v2/", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/" {
			http.Redirect(writer, request, openapi, 302)
			return
		}
		http.StripPrefix("/v2", handlers.CompressHandler(allowCORS(wsproxy.WebsocketProxy(gwmux))))
	})
	group.Go(func() error {
		return http.ListenAndServe(addrApi, mux)
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
}

func serveOpenAPI(prefix string, mux *http.ServeMux) error {
	_ = mime.AddExtensionType(".svg", "image/svg+xml")

	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	// Expose files in static on <host>/v2/openapi-ui
	fileServer := http.FileServer(statikFS)
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
	return nil
}

func unaryTimeoutInterceptor(timeout time.Duration) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		withTimeout, _ := context.WithTimeout(ctx, timeout)
		return handler(withTimeout, req)
	}
}

func parseStatus(s *status.Status) (string, map[string]string) {
	codeString := strconv.Itoa(runtime.HTTPStatusFromCode(s.Code()))
	dataString := map[string]string{}
	details := s.Details()
	if len(details) == 0 {
		return codeString, dataString
	}

	detail, ok := details[0].(*_struct.Struct)
	if !ok {
		return codeString, dataString
	}

	data := detail.AsMap()
	for k, v := range data {
		dataString[k] = fmt.Sprintf("%s", v)
	}
	code, ok := detail.GetFields()["code"]
	if ok {
		codeString = code.GetStringValue()
	}
	return codeString, dataString
}

func httpError(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	const fallback = `{"error": {"code": "500", "message": "failed to marshal error message"}}`

	contentType := marshaler.ContentType()
	w.Header().Set("Content-Type", contentType)

	s, ok := status.FromError(err)
	if !ok {
		s = status.New(codes.Unknown, err.Error())
	}
	st := runtime.HTTPStatusFromCode(s.Code())
	w.WriteHeader(st)

	codeString, data := parseStatus(s)
	delete(data, "code")

	jErr := json.NewEncoder(w).Encode(gw.ErrorBody{
		Error: &gw.ErrorBody_Error{
			Code:    codeString,
			Message: s.Message(),
			Data:    data,
		},
	})

	if jErr != nil {
		grpclog.Infof("Failed to write response: %v", err)
		w.Write([]byte(fallback))
	}
}

func requestExtractorFields() grpc_ctxtags.Option {
	return grpc_ctxtags.WithFieldExtractorForInitialReq(func(fullMethod string, req interface{}) map[string]interface{} {
		retMap := make(map[string]interface{})
		marshal, _ := json.Marshal(req)
		_ = json.Unmarshal(marshal, &retMap)
		if len(retMap) == 0 {
			return nil
		}
		return retMap
	})
}

type kitLogger struct {
	log.Logger
}

func (l *kitLogger) Log(keyvals ...interface{}) error {
	l.Info("API", keyvals...)
	return nil
}
