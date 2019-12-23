package v2

import (
	"context"
	"fmt"
	gw "github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/api/v2/service"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"net/http"
)

func Run(srvc *service.Service, addr string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()

	if err := gw.RegisterHttpServiceHandlerServer(ctx, mux, srvc); err != nil {
		return err
	}

	mux.Handle("GET", runtime.MustPattern(runtime.NewPattern(1, []int{2, 0}, []string{"subscribe"}, "", runtime.AssumeColonVerbOpt(true))), srvc.Subscribe)

	fmt.Println("listening")

	if err := http.ListenAndServe(addr, mux); err != nil {
		return err
	}

	return nil
}
