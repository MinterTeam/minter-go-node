module github.com/MinterTeam/minter-go-node

go 1.16

require (
	github.com/MinterTeam/node-grpc-gateway v1.4.3-0.20210816153334-49a13e269a0b
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/c-bata/go-prompt v0.2.3
	github.com/cosmos/iavl v0.15.3
	github.com/go-kit/kit v0.10.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/handlers v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.5.0
	github.com/marcusolsson/tui-go v0.4.0
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pkg/term v0.0.0-20190109203006-aa71e9d9e942 // indirect
	github.com/prometheus/client_golang v1.8.0
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/tendermint/go-amino v0.15.1
	github.com/tendermint/tendermint v0.34.10
	github.com/tendermint/tm-db v0.6.4
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802
	github.com/urfave/cli/v2 v2.0.0
	golang.org/x/crypto v0.0.0-20201117144127-c1f2f97bffc9
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/tendermint/tendermint => github.com/MinterTeam/tendermint v0.34.11-0.20210615080504-44952755291a
