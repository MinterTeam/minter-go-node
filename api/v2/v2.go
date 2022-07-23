package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MinterTeam/minter-go-node/api/v2/service"
	gw "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/MinterTeam/node-grpc-gateway/docs"
	kit_log "github.com/go-kit/kit/log"
	"github.com/gorilla/handlers"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/kit"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	_struct "google.golang.org/protobuf/types/known/structpb"
)

// Run initialises gRPC and API v2 interfaces
func Run(srv *service.Service, addrGRPC, addrAPI string, logger log.Logger) error {
	lis, err := net.Listen("tcp", addrGRPC)
	if err != nil {
		return err
	}

	unaryServerInterceptors := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(),
		unaryTimeoutInterceptor(srv.TimeoutDuration()),
	}
	streamServerInterceptors := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(),
	}

	if srv.EnabledPrometheus() {
		streamServerInterceptors = append(streamServerInterceptors, grpc_prometheus.StreamServerInterceptor)
		unaryServerInterceptors = append(unaryServerInterceptors, grpc_prometheus.UnaryServerInterceptor)

	}
	if srv.EnabledLogger() {
		kitLogger := &kitLogger{logger}
		option := kit.WithLevels(func(code codes.Code, logger kit_log.Logger) kit_log.Logger { return logger })
		streamServerInterceptors = append(streamServerInterceptors, grpc_ctxtags.StreamServerInterceptor(requestExtractorFields()), kit.StreamServerInterceptor(kitLogger, option))
		unaryServerInterceptors = append(unaryServerInterceptors, grpc_ctxtags.UnaryServerInterceptor(requestExtractorFields()), kit.UnaryServerInterceptor(kitLogger, option))
	}

	grpcServer := grpc.NewServer(
		grpc_middleware.WithStreamServerChain(streamServerInterceptors...),
		grpc_middleware.WithUnaryServerChain(unaryServerInterceptors...),
	)

	gw.RegisterApiServiceServer(grpcServer, srv)
	if srv.EnabledPrometheus() {
		grpc_prometheus.Register(grpcServer)
	}

	var group errgroup.Group

	group.Go(func() error {
		return grpcServer.Serve(lis)
	})

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	gwmux := runtime.NewServeMux(
		runtime.WithErrorHandler(httpError),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1000000000)),
	}
	err = gw.RegisterApiServiceHandlerFromEndpoint(ctx, gwmux, addrGRPC, opts)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	const openapi = "/v2/openapi-ui/"
	_ = serveOpenAPI(openapi, mux)
	mux.HandleFunc("/v2/", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/" {
			http.Redirect(writer, request, openapi, 302)
			return
		}
		if !strings.Contains(srv.Version(), "testnet") && request.URL.Path == "/v2/test/block" {
			http.Error(writer, "only testnet mode", http.StatusMethodNotAllowed)
			return
		}
		http.StripPrefix("/v2", handlers.CompressHandler(allowCORS(wsproxy.WebsocketProxy(gwmux)))).ServeHTTP(writer, request)
	})

	mux.Handle("/v2/custom/", http.StripPrefix("/v2/custom", srv.CustomHandlers()))

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

	// Expose files in static on <host>/v2/openapi-ui
	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.FS(docs.FS))))
	return nil
}

func unaryTimeoutInterceptor(timeout time.Duration) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		withTimeout, cencel := context.WithTimeout(ctx, timeout)
		defer cencel()
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
