# Minter Node

Minter is a blockchain network that lets people, projects, and companies issue and manage their own coins and trade them at a fair market price with absolute and instant liquidity.

[![version](https://img.shields.io/github/tag/MinterTeam/minter-go-node.svg)](https://github.com/MinterTeam/minter-go-node/releases/latest)
[![Go version](https://img.shields.io/badge/go-1.10-blue.svg)](https://github.com/moovweb/gvm)
[![license](https://img.shields.io/github/license/MinterTeam/minter-go-node.svg)](https://github.com/MinterTeam/minter-go-node/blob/master/LICENSE)

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

Install [Tendermint 0.20](https://github.com/tendermint/tendermint/blob/master/docs/install.rst)

```bash
$ mkdir $GOPATH/src/github.com/MinterTeam
$ cd $GOPATH/src/github.com/MinterTeam
$ git clone https://github.com/MinterTeam/minter-go-node.git

$ cd minter-go-node
$ make get_tools
$ make get_vendor_deps

$ make install

$ mkdir -p ~/.tendermint/data
$ mkdir -p ~/.minter/data

$ cp -R networks/testnet/ ~/.tendermint/config
```

Run Tendermint
```bash
$ tendermint node
```

Run Minter

```bash
$ minter
```