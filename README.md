# Minter Node

Minter is a blockchain network that lets people, projects, and companies issue and manage their own coins and trade them at a fair market price with absolute and instant liquidity.

[![version](https://img.shields.io/github/tag/MinterTeam/minter-go-node.svg)](https://github.com/MinterTeam/minter-go-node/releases/latest)
[![Go version](https://img.shields.io/badge/go-1.10-blue.svg)](https://github.com/moovweb/gvm)
[![license](https://img.shields.io/github/license/MinterTeam/minter-go-node.svg)](https://github.com/MinterTeam/minter-go-node/blob/master/LICENSE)
[![last-commit](https://img.shields.io/github/last-commit/MinterTeam/minter-go-node.svg)](https://github.com/MinterTeam/minter-go-node/commits/master)


_NOTE: This is alpha software. Please contact us if you intend to run it in production._

## Run using Docker

You'll need [docker](https://docker.com/) and [docker compose](https://docs.docker.com/compose/) installed.

Clone Minter to your machine
```bash
$ git clone https://github.com/MinterTeam/minter-go-node.git
$ cd minter-go-node
```

Prepare configs
```bash
$ mkdir -p ~/.tendermint/data
$ mkdir -p ~/.minter/data

$ chmod -R 0777 ~/.tendermint
$ chmod -R 0777 ~/.minter

$ cp -R networks/testnet/ ~/.tendermint/config
```

Start Minter
```bash
$ docker-compose up
```

## Build and run manually

You'll need **go** [installed](https://golang.org/doc/install) and the required
[environment variables set](https://github.com/tendermint/tendermint/wiki/Setting-GOPATH)

1. Install [Tendermint 0.20](https://github.com/tendermint/tendermint/blob/master/docs/install.rst)

2. Clone Minter to your machine
```bash
$ mkdir $GOPATH/src/github.com/MinterTeam
$ cd $GOPATH/src/github.com/MinterTeam
$ git clone https://github.com/MinterTeam/minter-go-node.git
$ cd minter-go-node
```

3. Get Tools & Dependencies

```bash
$ make get_tools
$ make get_vendor_deps
```

4. Compile
```bash
$ make install
```

5. Create data directories
```bash
$ mkdir -p ~/.tendermint/data
$ mkdir -p ~/.minter/data
```

6. Copy config and genesis file
```bash
$ cp -R networks/testnet/ ~/.tendermint/config
```

7. Run Tendermint
```bash
$ tendermint node
```

8. Run Minter

```bash
$ minter
```

## Troubleshooting

If you see error like this: 

```
ERROR: Failed to create node: Error starting proxy app connections: Error on replay: Wrong Block.Header.AppHash.  Expected 6D94BF43BB6C83F396FD8310BC2983F08C658344F9F348BB6675D1E5913230B3, got A2F322A4891092C690F5F0B80C1B9D5017A703035B63385108628EC244ECB191 
```

then your build of Minter Node and network build of Minter Node are different.