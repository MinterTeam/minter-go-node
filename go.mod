module github.com/MinterTeam/minter-go-node

go 1.17

require (
	github.com/MinterTeam/node-grpc-gateway v1.5.2-0.20220221151101-891c57b18d5b
	github.com/btcsuite/btcd v0.22.0-beta
	github.com/c-bata/go-prompt v0.2.5
	github.com/cosmos/cosmos-sdk v0.44.5
	github.com/cosmos/iavl v0.17.3
	github.com/go-kit/kit v0.12.0
	github.com/gogo/protobuf v1.3.3
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/handlers v1.5.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.1
	github.com/marcusolsson/tui-go v0.4.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.34.15
	github.com/tendermint/tm-db v0.6.6
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802
	github.com/urfave/cli/v2 v2.0.0
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
)

//replace github.com/cosmos/cosmos-sdk => github.com/klim0v/cosmos-sdk
replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

require (
	github.com/DataDog/zstd v1.4.5 // indirect
	github.com/Workiva/go-datastructures v1.0.53 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/confio/ics23/go v0.6.6 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/badger/v2 v2.2007.2 // indirect
	github.com/dgraph-io/ristretto v0.0.3 // indirect
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gdamore/encoding v0.0.0-20151215212835-b23993cbb635 // indirect
	github.com/gdamore/tcell v1.1.0 // indirect
	github.com/go-kit/log v0.2.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/orderedcode v0.0.1 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/gtank/merlin v0.1.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/lib/pq v1.10.4 // indirect
	github.com/libp2p/go-buffer-pool v0.0.2 // indirect
	github.com/lucasb-eyer/go-colorful v0.0.0-20180709185858-c7842319cf3a // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mimoo/StrobeGo v0.0.0-20181016162300-f8f6d4d2b643 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.4.2 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/petermattis/goid v0.0.0-20180202154549-b0b1615b78e5 // indirect
	github.com/pkg/term v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/rs/cors v1.8.0 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/sasha-s/go-deadlock v0.2.1-0.20190427202633-1595213edefa // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	golang.org/x/net v0.0.0-20211208012354-db4efeb81f4b // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211207154714-918901c715cf // indirect
	gopkg.in/ini.v1 v1.63.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/tendermint/tendermint => github.com/MinterTeam/tendermint v0.34.11-0.20211217161414-dda5434727d3

//replace github.com/tendermint/tendermint => /Users/klim0v/GolandProjects/tendermint
