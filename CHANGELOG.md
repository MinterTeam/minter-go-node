# Changelog

## 0.0.5
*June 29th, 2018*

BREAKING CHANGES

- [core] Remove Reserve Coin from coin object. All coins should be reserved with base coin.
- [testnet] New testnet chain id (minter-test-network-8)

IMPROVEMENT

- [api] Fix issue with not found coins

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
