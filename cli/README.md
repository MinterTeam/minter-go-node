# Node Command Line Interface

```
$ CGO_ENABLED=1 go run -tags "minter gcc" ./cmd/minter/main.go  manager --help
#or
$ ./node  manager --help
```
```text
COMMANDS:
   dial_peer, dp     connect a new peer
   prune_blocks, pb  delete block information
   status, s         display the current status of the blockchain
   net_info, ni      display network data
   exit, e           exit
   help, h           Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help (default: false)
   --version, -v  print the version (default: false)
```