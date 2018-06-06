# Minter Node

Minter is a blockchain network that lets people, projects, and companies issue and manage their own coins and trade them at a fair market price with absolute and instant liquidity.

## Run using Docker

```
git clone https://github.com/MinterTeam/minter-go-node.git $GOPATH/src/minter
cd $GOPATH/src/minter
make get_tools
make get_vendor_deps
make build-linux
make build-docker
docker-compose up
```
