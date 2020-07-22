module github.com/MinterTeam/minter-go-node

go 1.13

require (
	github.com/MinterTeam/node-grpc-gateway v1.1.3-0.20200721223527-f6c600c7be2f
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/c-bata/go-prompt v0.2.3
	github.com/go-kit/kit v0.10.0
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.1-0.20190118093823-f849b5445de4
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/marcusolsson/tui-go v0.4.0
	github.com/mattn/go-tty v0.0.3 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pkg/term v0.0.0-20190109203006-aa71e9d9e942 // indirect
	github.com/prometheus/client_golang v1.5.1
	github.com/rakyll/statik v0.1.7
	github.com/rs/cors v1.7.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.6.3
	github.com/stretchr/testify v1.6.1
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d
	github.com/tendermint/go-amino v0.15.1
	github.com/tendermint/iavl v0.14.0
	github.com/tendermint/tendermint v0.33.6
	github.com/tendermint/tm-db v0.5.1
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5
	github.com/urfave/cli/v2 v2.0.0
	gitlab.com/tslocum/cview v1.4.7
	golang.org/x/crypto v0.0.0-20200429183012-4b2356b1ed79
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20200523222454-059865788121
	google.golang.org/grpc v1.30.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15
)

replace (
	github.com/MinterTeam/node-grpc-gateway => /Users/klim0v/GolandProjects/node-grpc-gateway
)