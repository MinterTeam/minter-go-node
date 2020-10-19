package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/service"
	gw "github.com/MinterTeam/node-grpc-gateway/api_pb"
	_ "github.com/MinterTeam/node-grpc-gateway/statik"
	kit_log "github.com/go-kit/kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/gorilla/handlers"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/kit"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rakyll/statik/fs"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	_struct "google.golang.org/protobuf/types/known/structpb"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Run initialises gRPC and API v2 interfaces
func Run(srv *service.Service, addrGRPC, addrAPI string, logger log.Logger) error {
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
	runtime.WithErrorHandler(httpError)

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
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true, // todo
			},
		}),
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
	// if strings.Contains(srv.Version(), "testnet") { // todo: uncomment to prod
	gwmux.Handle("GET", runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1}, []string{"test", "block"}, "", runtime.AssumeColonVerbOpt(true))), registerTestHandler(context.Background(), gwmux, srv))
	// }
	mux.HandleFunc("/v2/", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/" {
			http.Redirect(writer, request, openapi, 302)
			return
		}
		http.StripPrefix("/v2", handlers.CompressHandler(allowCORS(wsproxy.WebsocketProxy(gwmux)))).ServeHTTP(writer, request)
	})
	group.Go(func() error {
		return http.ListenAndServe(addrAPI, mux)
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
	body := &gw.ErrorBody{}
	w.Header().Set("Content-Type", marshaler.ContentType(body))

	s, ok := status.FromError(err)
	if !ok {
		s = status.New(codes.Unknown, err.Error())
	}
	st := runtime.HTTPStatusFromCode(s.Code())
	w.WriteHeader(st)

	codeString, data := parseStatus(s)
	delete(data, "code")

	body.Error = &gw.ErrorBody_Error{
		Code:    codeString,
		Message: s.Message(),
		Data:    data,
	}

	buf, merr := marshaler.Marshal(body)
	if merr != nil {
		grpclog.Infof("Failed to marshal error message %q: %v", s, merr)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, `{"error": {"code": "500", "message": "failed to marshal error message"}}`); err != nil {
			grpclog.Infof("Failed to write response: %v", err)
		}
		return
	}
	if _, err := w.Write(buf); err != nil {
		grpclog.Infof("Failed to write response: %v", err)
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

func registerTestHandler(ctx context.Context, mux *runtime.ServeMux, client *service.Service) runtime.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		resp, md, err := requestTestBlock(rctx, inboundMarshaler, client, req, pathParams)
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		runtime.ForwardResponseMessage(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	}
}

func requestTestBlock(ctx context.Context, _ runtime.Marshaler, server *service.Service, _ *http.Request, _ map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq empty.Empty
	var metadata runtime.ServerMetadata

	msg, err := server.TestBlock(ctx, &protoReq)
	return msg, metadata, err

}
