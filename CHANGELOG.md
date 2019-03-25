# Changelog

## 0.15.0

BREAKING CHANGES

- [tendermint] Update to [v0.31.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0310) 

IMPROVEMENT

- [invariants] Add invariants checker each 720 blocks
- [core] Delete coins with 0 reserves #217
- [genesis] Add option to export/import state
- [api] Add ?include_stakes to /candidates endpoint #222
- [api] Change `stake` to `value` in DelegateTx
- [api] Change `pubkey` to `pub_key` in all API resources and requests
- [events] Add CoinLiquidation event #221
- [mempool] Recheck mempool once per minute

BUG FIXES

- [core] Fix double sign slashing issue #215
- [core] Fix issue with slashing small stake #209
- [core] Fix coin creation issue
- [core] Fix mempool issue #220
- [api] Make block hash lowercase #214

## 0.14.3

BUG FIXES

- [core] Temp fix for consensus failure

## 0.14.2

BUG FIXES

- [events] Fix slash event on double sign (full resync needed)

## 0.14.1

IMPROVEMENT

- [api] Add /addresses endpoint
- [api] Add evidence data to /block
- [tendermint] Update to [v0.30.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0301) 

BUG FIXES

- [api] Fix /block endpoint

## 0.13.1

BUG FIXES

- [core] Fix sync issue

## 0.13.0

BREAKING CHANGES

- [tendermint] Update to [v0.30.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0300) 

BUG FIXES

- [core] Fix max tx length

## 0.12.1

BUG FIXES

- [core] Fix "No info about LastBlocksTimeDelta is available" issue

## 0.12.0

BREAKING CHANGES

- [core] Updated commission handling
- [core] Fix multisend issue
- [core] Extend max tx size
- [api] Fixes in error responses

## 0.11.0

BREAKING CHANGES

- [core] Fix coin convert issue
- [tendermint] Update to [v0.29.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0291) 

## 0.10.1
*Jan 22th, 2019*

BREAKING CHANGES

- [tendermint] Update to [v0.29.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0290) 

## 0.10.0
*Jan 20th, 2019*

BREAKING CHANGES

- [core] Add EditCandidate transaction
- [core] Make validators count logic conforms to mainnet
- [tendermint] Update to [v0.28.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0281) 

BUG FIXES

- [core] Various bug fixes

IMPROVEMENT

- [mempool] Add variable min gas price threshold
- [p2p] Lower FlushThrottleTimeout to 10 ms
- [rpc] RPC errors are now delivered with 500 HTTP code
- [rpc] Prettify RPC errors

## 0.9.6
*Dec 27th, 2018*

BUG FIXES

- [core] Fix issue with corrupted db

## 0.9.5
*Dec 26th, 2018*

BUG FIXES

- [core] Fix issue with corrupted db

## 0.9.4
*Dec 26th, 2018*

IMPROVEMENT

- [mempool] Disable tx rechecking

BUG FIXES

- [core] Fix issue with bag tx occupying mempool

## 0.9.3
*Dec 25th, 2018*

BUG FIXES

- [core] Fix sell all coin tx

## 0.9.2
*Dec 25th, 2018*

BUG FIXES

- [core] Increase max block bytes

## 0.9.1
*Dec 24th, 2018*

BUG FIXES

- [api] Fix create coin tx error

## 0.9.0
*Dec 24th, 2018*

IMPROVEMENT

- [events] Refactor events
- [api] #183 Report if node has full state history in /status
- [api] #164 Add /unconfirmed_txs endpoint
- [api] Add /max_gas endpoint
- [core] Do not accept 2 transactions from same address in mempool at once
- [core] Add missing tags to transactions
- [core] Dynamically adjust max gas in blocks
- [core] Update commissions
- [tendermint] Update to v0.27.4

BUG FIXES

- [core] Fix issue with `SellAll` tx
- [core] Fix issue #182 with candidate owner's address
- [core] Fix max coin supply
- [api] Fix tx tags

## 0.8.5
*Dec 11th, 2018*

BUG FIXES

- [api] Fix estimate coin buy empty response
- [api] Set quotes as not necessary attribute

## 0.8.4
*Dec 10th, 2018*

BUG FIXES

- [core] Fix tx processing bug

## 0.8.3
*Dec 10th, 2018*

BUG FIXES

- [events] Fix pub key formatting in API

## 0.8.2
*Dec 10th, 2018*

BUG FIXES

- [log] Add json log format

## 0.8.1
*Dec 10th, 2018*

IMPROVEMENT

- [core] Speed-up tx processing

BUG FIXES

- [config] Change default seed node

## 0.8.0
*Dec 3rd, 2018*

BREAKING CHANGES

- [api] Switch to RPC protocol
- [api] Separate events from block in API
- [core] Fix issue with incorrect coin conversion
- [core] Limit coins supply to 1,000,000,000,000,000
- [core] Set minimal reserve and min/max coin supply in CreateCoin tx
- [core] Add MinimumValueToBuy and MaximumValueToSell to convert transactions
- [tendermint] Update to [v0.27.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0270) 

IMPROVEMENT

- [logs] Add `log_format` option to config
- [events] Add UnbondEvent

## 0.7.6
*Nov 27th, 2018*

IMPROVEMENT

- [tendermint] Update to [v0.26.4](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0264) 

BUG FIXES

- [node] Fix issue #168 with unexpected database corruption

## 0.7.5
*Nov 22th, 2018*

BUG FIXES

- [api] Fix issue in which transaction appeared in `/api/transaction` before actual execution

## 0.7.4
*Nov 20th, 2018*

BUG FIXES

- [tendermint] "Send failed" is logged at debug level instead of error
- [tendermint] Set connection config properly instead of always using default
- [tendermint] Seed mode fixes:
   - Only disconnect from inbound peers
   - Use FlushStop instead of Sleep to ensure all messages are sent before disconnecting

## 0.7.3
*Nov 18th, 2018*

BUG FIXES

- [core] More fixes on issue with negative coin reserve

## 0.7.2
*Nov 18th, 2018*

BUG FIXES

- [core] Fix issue with negative coin reserve

## 0.7.1
*Nov 16th, 2018*

IMPROVEMENT
- [tendermint] Update to [v0.26.2](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0262) 

## 0.7.0
*Nov 15th, 2018*

BREAKING CHANGES

- [api] `/api/sendTransaction` is now returns only `checkTx` result. Applications are now forced to manually check if transaction is included in blockchain.
- [tendermint] Update to [v0.26.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0261) 
- [core] Block hash is now 32 bytes length

IMPROVEMENT

- [core] Add `MultisendTx`
- [core] Add special cases to Formulas #140
- [core] Stake unbond now instant after dropping of from 1,000st place #146
- [p2p] Default send and receive rates are now 15mB/s
- [mempool] Set max mempool size to 10,000txs
- [gui] Small GUI improvements

## 0.6.0
*Oct 30th, 2018*

BREAKING CHANGES

- [core] Set validators limit to 100 for testnet
- [core] SetCandidateOff transaction now applies immediately
- [tendermint] Update to [v0.26.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0260) 

IMPROVEMENT

- [config] Add keep_state_history option
- [api] Limit API requests

## 0.5.1
*Oct 22th, 2018*

BUG FIXES

- [core] Fixed bug with unexpected node backoff

## 0.5.0
*Oct 15th, 2018*

BREAKING CHANGES

- [core] Multisig wallets
- [core] Sub coin reserve and supply on slash
- [core] Change unbond time for testnet to 720 blocks
- [core] Limit candidates count to validatorsLimit*3 at given block
- [core] Limit delegators count to 1000 per candidate/validator

IMPROVEMENT

- [tendermint] Update to [v0.25.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0250) 

## 0.4.2
*Sept 21th, 2018*

BUG FIXES

- [api] Fix concurrent API calls

## 0.4.1
*Sept 20th, 2018*

IMPROVEMENT

- [core] Speed up synchronization 

BUG FIXES

- [gui] Fix validator status

## 0.4.0
*Sept 18th, 2018*

BREAKING CHANGES

- [core] Switch Ethereum Patricia Tree to IAVL
- [core] Change consensus TimeoutCommit to 4.5 sec, TimeoutPropose to 2 sec
- [core] Now validator punished if it misses 12 of 24 last blocks

IMPROVEMENT

- [config] Add validator mode
- [api] Include events by default
- [gui] Add validator status

## 0.3.8
*Sept 17th, 2018*

BUG FIXES

- [core] Proper handle of db errors

## 0.3.7
*Sept 17th, 2018*

IMPROVEMENT

- [core] Performance update

## 0.3.6
*Sept 15th, 2018*

BUG FIXES

- [core] Critical fix

## 0.3.5
*Sept 13th, 2018*

IMPROVEMENT

- [api] Add Code and Log fields in transaction api

## 0.3.4
*Sept 13th, 2018*

IMPROVEMENT

- [api] Optimize events. WARNING! If you are using events you should re-sync blockchain from scratch.
- [api] Refactor api

## 0.3.3
*Sept 8th, 2018*

IMPROVEMENT

- [api] Add block size in bytes
- [api] #100 Add "events" to block response. To get events add ?withEvents=true to request URL. 
WARNING! You should sync blockchain from scratch to get this feature working

## 0.3.2
*Sept 8th, 2018*

BUG FIXES

- [core] Fix null pointer exception

## 0.3.1
*Sept 8th, 2018*

BUG FIXES

- [core] Fix shutdown issue

## 0.3.0
*Sept 8th, 2018*

BREAKING CHANGES

- [core] Validators are now updated each 120 blocks
- [core] Validators are now updated then at least one of current validators exceed 12 missed blocks 
- [tendermint] Update Tendermint to v0.24.0

IMPROVEMENT

- [p2p] Add seed nodes
- [sync] Speed up synchronization
- [core] Extend max payload size to 1024 bytes
- [core] Add network id checker
- [core] Add tx.sell_amount to SellAllCoin tags
- [core] Change punishment for byzantine behavior
- [api] Limit balance watchers to 10 clients
- [config] Add config file
- [config] Add GUI listen address to config
- [config] Add API listen address to config
- [docs] Update documentation
- [validators] Remove 0-valued stakes from state

BUG FIXES

- [core] Fix issue #77 Incorrect createCoin fee
- [core] Fix issue with insufficient coin reserve in buy coin tx
- [core] Fix unbond transaction
- [api] Fix issue #82

## 0.2.4
*Aug 24th, 2018*

BUG FIXES

- [api] Fix estimateTxCommission endpoint

IMPROVEMENT

- [gui] Minor GUI updates

## 0.2.2
*Aug 23th, 2018*

In this update we well test blockchain's hardfork.
There is no need to wipe old data, just be sure to update binary
until 15000 block.

BUG FIXES

- [validators] Fix api

## 0.2.1
*Aug 23th, 2018*

In this update we well test blockchain's hardfork.
There is no need to wipe old data, just be sure to update binary
until 15000 block.

BUG FIXES

- [validators] Fix validators issue

## 0.2.0
*Aug 22th, 2018*

BREAKING CHANGES

- [testnet] New testnet id
- [core] New rewards
- [core] Validators list are now updated each 12 blocks
- [core] Set DAO commission to 10% 
- [core] Add Developers commission of 10%
- [core] Now stake of custom coin is calculated by selling all such staked coins
- [api] Reformatted candidates and validators endpoints
- [api] tx.return tags are now encoded as strings

IMPROVEMENT

- [tendermint] Update tendermint to 0.23.0
- [api] Add block reward to api
- [api] Add bip_value field to Stake
- [api] Add /api/candidates endpoint
- [api] Add /api/estimateTxCommission endpoint
- [gui] Minor GUI update

## 0.1.9
*Aug 19th, 2018*

BUG FIXES
- [core] Critical fix

## 0.1.8
*Aug 4th, 2018*

BUG FIXES
- [core] Critical fix

## 0.1.7
*Jule 30th, 2018*

BREAKING CHANGES

- [testnet] New testnet id

IMPROVEMENT

- [validators] Added flag ``--reset-private-validator``
- [testnet] Main validator stake is set to 1 mln MNT by default

## 0.1.6
*Jule 30th, 2018*

BREAKING CHANGES

- [testnet] New testnet id

BUG FIXES

- [core] Fixed critical bug

## 0.1.5
*Jule 28th, 2018*

BUG FIXES

- [tendermint] Update tendermint to 0.22.8
- [core] Temporary critical fix

## 0.1.4
*Jule 25th, 2018*

IMPROVEMENT

- [tendermint] Update tendermint to 0.22.6

## 0.1.3
*Jule 25th, 2018*

IMPROVEMENT

- [tendermint] Update tendermint to 0.22.5

## 0.1.0
*Jule 23th, 2018*

BREAKING CHANGES

- [core] 0.1x transaction fees
- [core] Genesis is now encapsulated in code
- [core] Add new transaction type: SellAllCoin
- [core] Add GasCoin field to transaction
- [config] New config directories
- [api] Huge API update. For more info see docs

IMPROVEMENT

- [binary] Now Minter is available as single binary. There is no need to install Tendermint
- [config] 10x default send/recv rate
- [config] Recheck after empty blocks
- [core] Check transaction nonce before adding to mempool
- [performance] Huge performance enhancement due to getting rid of network overhead between tendermint and minter
- [gui] GUI introduced! You can use it by visiting http://localhost:3000/ in your local browser

BUG FIXES

- [api] Fixed raw transaction output

## 0.0.6
*Jule 16th, 2018*

BREAKING CHANGES

- [core] Change commissions
- [testnet] New testnet id
- [core] Fix transaction decoding issue
- [core] Remove transaction ConvertCoin, add SellCoin and BuyCoin. For details see the docs.
- [core] Coin name is now limited to max 64 bytes
- [api] Update estimate exchange endpoint

IMPROVEMENT

- [api] Update transaction api
- [api] Add transaction result to block api
- [mempool] Mempool cache is disabled
- [tendermint] Updated to v0.22.4
- [versioning] Adapt Semantic Versioning https://semver.org/
- [client] Add --disable-api flag to client

## 0.0.5
*Jule 4rd, 2018*

BREAKING CHANGES

- [core] Remove Reserve Coin from coin object. All coins should be reserved with base coin
- [core] Limit tx payload and service data to 128 bytes
- [core] Fix critical issue with instant convert of 2 custom coins 
- [testnet] New testnet chain id (minter-test-network-9)
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
