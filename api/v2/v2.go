package v2

import (
	"context"
	"fmt"
	gw "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
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

	fmt.Println("listening")

	if err := http.ListenAndServe(addr, mux); err != nil {
		return err
	}

	return nil
}
