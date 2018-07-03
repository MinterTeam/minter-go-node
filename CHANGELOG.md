# Changelog

## 0.0.5
*Jule 3rd, 2018*

BREAKING CHANGES

- [core] Remove Reserve Coin from coin object. All coins should be reserved with base coin
- [core] Limit tx payload and service data to 128 bytes
- [testnet] New testnet chain id (minter-test-network-8)
- [tendermint] Switched to v0.22.0

IMPROVEMENT

- [api] Fix issue with not found coins

BUG FIXES

- [api] Fix transaction endpoint

## 0.0.4

*June 24th, 2018*

BREAKING CHANGES

- [validators] Reward now is payed each 12 blocks
- [validators] Change total "validators' power" to 100 mln
- [tendermint] Switched to v0.21.0
- [testnet] New testnet chain id
- [api] Changed */api/block* response format

IMPROVEMENT

- [docs] Updated docs

BUG FIXES

- [validators] Fixed issue with incorrect pubkey length
