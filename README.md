# Minter Node

Minter is a blockchain network that lets people, projects, and companies issue and manage their own coins and trade them at a fair market price with absolute and instant liquidity.

[![version](https://img.shields.io/github/tag/MinterTeam/minter-go-node.svg)](https://github.com/MinterTeam/minter-go-node/releases/latest)
[![Go version](https://img.shields.io/badge/go-1.10-blue.svg)](https://github.com/moovweb/gvm)
[![license](https://img.shields.io/github/license/MinterTeam/minter-go-node.svg)](https://github.com/MinterTeam/minter-go-node/blob/master/LICENSE)

_NOTE: This is alpha software. Please contact us if you intend to run it in production._

## Run using Docker

---

You'll need **go** [installed](https://golang.org/doc/install) and the required [environment variables](https://github.com/tendermint/tendermint/wiki/Setting-GOPATH) set.

---

Clone Minter to your machine

```
git clone https://github.com/MinterTeam/minter-go-node.git $GOPATH/src/minter
cd $GOPATH/src/minter
```

Install dependencies

```
make get_tools
make get_vendor_deps
```

Build docker image
```
make build-linux
make build-docker
```

Prepare configs
```
mkdir -p ~/.tendermint/data
mkdir -p ~/.minter/data

chmod -R 0777 ~/.tendermint
chmod -R 0777 ~/.minter

cp -R networks/testnet/ ~/.tendermint/config
```

Start Minter
```
docker-compose up
```
