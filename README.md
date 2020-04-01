<p align="center" background="black"><img src="minter-logo.svg" width="400"></p>

<p align="center">
<a href="https://github.com/MinterTeam/minter-go-node/releases/latest"><img src="https://img.shields.io/github/tag/MinterTeam/minter-go-node.svg" alt="Version"></a>
<a href="https://github.com/moovweb/gvm"><img src="https://img.shields.io/badge/go-1.10-blue.svg" alt="Go version"></a>
<a href="https://github.com/MinterTeam/minter-go-node/blob/master/LICENSE"><img src="https://img.shields.io/github/license/MinterTeam/minter-go-node.svg" alt="License"></a>
<a href="https://github.com/MinterTeam/minter-go-node/commits/master"><img src="https://img.shields.io/github/last-commit/MinterTeam/minter-go-node.svg" alt="Last commit"></a>
<a href="https://goreportcard.com/report/github.com/MinterTeam/minter-go-node"><img src="https://goreportcard.com/badge/github.com/MinterTeam/minter-go-node" alt="Go Report Card"></a>
<a href="https://github.com/MinterTeam/minter-go-node/actions"><img src="https://github.com/MinterTeam/minter-go-node/workflows/docker/badge.svg" alt="Github Actions Report Card"></a>
<a href="https://hub.docker.com/r/minterteam/minter"><img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/minterteam/minter"></a>
</p>

Minter is a blockchain network that lets people, projects, and companies issue and manage their own coins and trade them at a fair market price with absolute and instant liquidity.

_NOTE: This is alpha software. Please contact us if you intend to run it in production._

## Installation
### Docker
1. Grab latest docker-compose, save <a href="https://raw.githubusercontent.com/MinterTeam/minter-go-node/master/docker-compose.yml">docker-compose.yml</a> and run ```docker-compose up -d```. 
To run it in production we recommend to use bind host mount instead of volume. 

2. To build from source clone this repo, make your changes and run ```docker-compose up --build -d```

### Manual
You can get official installation instructions in our [docs](https://docs.minter.network/#section/Install-Minter).

1. Download Minter Node

    Get [latest binary build](https://github.com/MinterTeam/minter-go-node/releases) suitable for your architecture and unpack it to desired folder.

2. Run Minter Node

```bash
./minter node
```

## Resources

- [Documentation](https://docs.minter.network)
- [Official site](https://minter.network)
- [About Minter Blockchain](https://about.minter.network)
- [Minter Console](https://console.minter.network)
- [Minter Explorer](https://explorer.minter.network/)
- [Telegram Bot Wallet](https://t.me/BipWallet_Bot)
- [Android Wallet](https://play.google.com/store/apps/details?id=network.minter.bipwallet)

### Related repositories

- [minter-go-node-docs](https://github.com/MinterTeam/minter-go-node-docs) - Docs for Minter Node
- [node-grpc-gateway](https://github.com/MinterTeam/node-grpc-gateway) - gRPC interface and Swagger for Node API v2

### Community
- [Telegram Channel (English)](https://t.me/MinterTeam)
- [Telegram Channel (Russian)](https://t.me/MinterNetwork)
- [Telegram Chat (English)](http://t.me/joinchat/EafyERJSJZJ-nwH_139jLQ)
- [Telegram Chat (Russian)](https://t.me/joinchat/EafyEVD-HEOxDcv8YyaqNg)

## Versioning

### SemVer

Minter uses [SemVer](http://semver.org/) to determine when and how the version changes.
According to SemVer, anything in the public API can change at any time before version 1.0.0

To provide some stability to Minter users in these 0.X.X days, the MINOR version is used
to signal breaking changes across a subset of the total public API. This subset includes all
interfaces exposed to other processes, but does not include the in-process Go APIs.

### Upgrades

In an effort to avoid accumulating technical debt prior to 1.0.0,
we do not guarantee that breaking changes (ie. bumps in the MINOR version)
will work with existing blockchain. In these cases you will
have to start a new blockchain, or write something custom to get the old
data into the new chain.

However, any bump in the PATCH version should be compatible with existing histories
(if not please open an [issue](https://github.com/MinterTeam/minter-go-node/issues)).
