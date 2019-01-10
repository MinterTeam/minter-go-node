Minter Node API
===============

Minter Node API is based on JSON format. JSON is a lightweight data-interchange format.
It can represent numbers, strings, ordered sequences of values, and collections of name/value pairs.

Status
^^^^^^

This endpoint shows current state of the node. You also can use it to check if node is running in
normal mode.

.. code-block:: bash

    curl -s 'localhost:8841/status'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "version": "0.8.0",
        "latest_block_hash": "171F3A749F85425147986DD90EA0C397440B6A3C1FEF8F30E5E5F729DA174CC2",
        "latest_app_hash": "55E75C9860E56AF3DEB8DD55741185F658569AB43C084436DDDB69CBFB06CC63",
        "latest_block_height": "4",
        "latest_block_time": "2018-12-03T13:18:42.50969Z",
        "state_history": "on",
        "tm_status": {
          "node_info": {
            "protocol_version": {
              "p2p": "4",
              "block": "7",
              "app": "0"
            },
            "id": "9d5eb9f8fb7ada3ff6228841c3500f39e3121901",
            "listen_addr": "tcp://0.0.0.0:26656",
            "network": "minter-test-network-27-local",
            "version": "0.26.4",
            "channels": "4020212223303800",
            "moniker": "MacBook-Pro-Daniil-2.local",
            "other": {
              "tx_index": "on",
              "rpc_address": "tcp://0.0.0.0:26657"
            }
          },
          "sync_info": {
            "latest_block_hash": "171F3A749F85425147986DD90EA0C397440B6A3C1FEF8F30E5E5F729DA174CC2",
            "latest_app_hash": "55E75C9860E56AF3DEB8DD55741185F658569AB43C084436DDDB69CBFB06CC63",
            "latest_block_height": "4",
            "latest_block_time": "2018-12-03T13:18:42.50969Z",
            "catching_up": false
          },
          "validator_info": {
            "address": "AB15A084DD592699812E9B22385C1959E7AEFFB8",
            "pub_key": {
              "type": "tendermint/PubKeyEd25519",
              "value": "4LpQ40aLB/u8EnhAlT649P5X1ugWLfk7rv159dW8K5c="
            },
            "voting_power": "100000000"
          }
        }
      }
    }

Max gas
^^^^^^^

This endpoint shows maximum gas value for given block

.. code-block:: bash

    curl -s 'localhost:8841/max_gas'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": "100000"
    }

Min gas price
^^^^^^^^^^^^^

This endpoint shows min acceptable gas price for tx to be included in mempool

.. code-block:: bash

    curl -s 'localhost:8841/min_gas_price'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": "1"
    }

Candidate
^^^^^^^^^

This endpoint shows candidate's info by provided public_key. It will respond with ``404`` code if candidate is not
found.

- **candidate_address** - Address of a candidate in minter network. This address is used to manage
  candidate and receive rewards.
- **total_stake** - Total stake calculated in base coin (MNT or BIP).
- **commission** - Commission for delerators. Measured in percents. Can be 0..100.
- **accumulated_reward** - Reward waiting to be sent to validator and his delegators. Reward is payed each 12 blocks.
- **stakes** - List of candidate's stakes.
- **created_at_block** - Height of block when candidate was created.
- **status** - Status of a candidate.

    - ``1`` - Offline
    - ``2`` - Online

- **absent_times** - How many blocks candidate missed. If this number reaches 12, then candidate's stake will be
  slashed by 1% and candidate will be turned off.

.. code-block:: bash

    curl -s 'localhost:8841/candidate?pubkey={public_key}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "reward_address": "Mxee81347211c72524338f9680072af90744333146",
        "owner_address": "Mxee81347211c72524338f9680072af90744333146",
        "total_stake": 0,
        "pub_key": "Mpe0ba50e3468b07fbbc127840953eb8f4fe57d6e8162df93baefd79f5d5bc2b97",
        "commission": "100",
        "stakes": [
          {
            "owner": "Mxee81347211c72524338f9680072af90744333146",
            "coin": "MNT",
            "value": "1000000000000000000000000",
            "bip_value": "1000000000000000000000000"
          }
        ],
        "created_at_block": "1",
        "status": 2
      }
    }

Validators
^^^^^^^^^^

Returns list of active validators.

.. code-block:: bash

    curl -s 'localhost:8841/validators'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": [
        {
          "pubkey": "Mpddfadfb15908ed5607c79e66aaf4030ef93363bd1846d64186d52424b1896c83",
          "voting_power": "100000000"
        }
      ]
    }

Address
^^^^^^^

Returns the balance of given account and the number of outgoing transaction.

.. code-block:: bash

    curl -s 'localhost:8841/address?address={address}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "balance": {
          "MNT": "100010489500000000000000000"
        },
        "transaction_count": "0"
      }
    }

| **Result->balance**: Map of balances. CoinSymbol => Balance (in pips).
| **Result->transaction_count**: Count of transactions sent from the account.

Send transaction
^^^^^^^^^^^^^^^^

Sends transaction to the Minter Network. **Note:** tx should start with 0x prefix.

.. code-block:: bash

    curl -s 'localhost:8841/send_transaction?tx={transaction}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "code": 0,
        "data": "",
        "log": "",
        "hash": "C6C6B5008AF8077FB0CE817DDB79268D1C66B6B353AF76778CA5A264A80069DB"
      }
    }

**Result**: Transaction hash.

Transaction
^^^^^^^^^^^

.. code-block:: bash

    curl -s 'localhost:8841/transaction?hash={hash}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "hash": "C6C6B5008AF8077FB0CE817DDB79268D1C66B6B353AF76778CA5A264A80069DB",
        "raw_tx": "f88701018a4d4e540000000000000001aae98a4d4e540000000000000094ee81347211c72524338f9680072af9074433314688a688906bd8b0000084546573748001b845f8431ba098fd9402b0af434f461eecdad89908655c779fb394b7624a0c37198f931f27a1a075e73a04f81e2204d88826ac851b2b3da359e4a9a16ac6c17e992fa0a3de0c48",
        "height": "387",
        "index": 0,
        "from": "Mxee81347211c72524338f9680072af90744333146",
        "nonce": "1",
        "gas_price": "1",
        "gas_coin": "MNT",
        "gas_used": "18",
        "type": 1,
        "data": {
          "coin": "MNT",
          "to": "Mxee81347211c72524338f9680072af90744333146",
          "value": "12000000000000000000"
        },
        "payload": "VGVzdA==",
        "tags": {
          "tx.coin": "MNT",
          "tx.type": "01",
          "tx.from": "ee81347211c72524338f9680072af90744333146",
          "tx.to": "ee81347211c72524338f9680072af90744333146"
        }
      }
    }

Block
^^^^^

Returns block data at given height.

.. code-block:: bash

    curl -s 'localhost:8841/block?height={height}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "hash": "0B1226C12783373BB2FFB451A104FF2BE47F59B8E7B6690B7712AADBA197D2FC",
        "height": "9",
        "time": "2018-12-05T09:14:57.114925Z",
        "num_txs": "1",
        "total_txs": "1",
        "transactions": [
          {
            "hash": "Mt0e765f48042683160d33c610a90845aeef5f8e0d71cab60e01895f8bd973d614",
            "raw_tx": "f8a701018a4d4e540000000000000006b84df84b94ee81347211c72524338f9680072af90744333146a021e1d043c6d9c0bb0929ab8d1dd9f3948de0f5ad7234ce773a501441d204aa9e0a8a4d4e5400000000000000888ac7230489e80000808001b845f8431ca0a7cfaf4ab3b64695380a5fd2f86f5fd29a56c722572dcb1a7fbc49ba8ff1cdc0a06be96fdf026ed7da605cfa1a606c134d99fea51717dbd57997e5e021ef714944",
            "from": "Mxee81347211c72524338f9680072af90744333146",
            "nonce": "1",
            "gas_price": "1",
            "type": 6,
            "data": {
              "address": "Mxee81347211c72524338f9680072af90744333146",
              "pub_key": "Mp21e1d043c6d9c0bb0929ab8d1dd9f3948de0f5ad7234ce773a501441d204aa9e",
              "commission": "10",
              "coin": "MNT",
              "stake": "10000000000000000000"
            },
            "payload": "",
            "service_data": "",
            "gas": "10000",
            "gas_coin": "MNT",
            "gas_used": "10000",
            "tags": {}
          }
        ],
        "block_reward": "333000000000000000000",
        "size": "1230",
        "proposer": "Mpddfadfb15908ed5607c79e66aaf4030ef93363bd1846d64186d52424b1896c83",
        "validators": [
          {
            "pubkey": "Mpddfadfb15908ed5607c79e66aaf4030ef93363bd1846d64186d52424b1896c83",
            "signed": true
          }
        ]
      }
    }

Events
^^^^^^

Returns events at given height.

.. code-block:: bash

    curl -s 'localhost:8841/events?height={height}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "events": [
          {
            "type": "minter/RewardEvent",
            "value": {
              "role": "DAO",
              "address": "Mxee81347211c72524338f9680072af90744333146",
              "amount": "367300000000000000000",
              "validator_pub_key": "Mp4d7064646661646662313539303865643536303763373965363661616634303330656639333336336264313834366436343138366435323432346231383936633833"
            }
          },
          {
            "type": "minter/RewardEvent",
            "value": {
              "role": "Developers",
              "address": "Mx444c4f1953ea170f74eabef4eee52ed8276a7d5e",
              "amount": "367300000000000000000",
              "validator_pub_key": "Mp4d7064646661646662313539303865643536303763373965363661616634303330656639333336336264313834366436343138366435323432346231383936633833"
            }
          },
          {
            "type": "minter/RewardEvent",
            "value": {
              "role": "Validator",
              "address": "Mxee81347211c72524338f9680072af90744333146",
              "amount": "2938400000000000000000",
              "validator_pub_key": "Mp4d7064646661646662313539303865643536303763373965363661616634303330656639333336336264313834366436343138366435323432346231383936633833"
            }
          }
        ]
      }
    }

Candidates
^^^^^^^^^^

Returns full list of candidates.

.. code-block:: bash

    curl -s 'localhost:8841/candidates?height={height}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": [
        {
          "reward_address": "Mxee81347211c72524338f9680072af90744333146",
          "owner_address": "Mxee81347211c72524338f9680072af90744333146",
          "total_stake": "1000000000000000000000000",
          "pubkey": "Mpddfadfb15908ed5607c79e66aaf4030ef93363bd1846d64186d52424b1896c83",
          "commission": "100",
          "created_at_block": "1",
          "status": 2
        },
        {
          "reward_address": "Mxee81347211c72524338f9680072af90744333146",
          "owner_address": "Mxee81347211c72524338f9680072af90744333146",
          "total_stake": "9900000000000000000",
          "pubkey": "Mp21e1d043c6d9c0bb0929ab8d1dd9f3948de0f5ad7234ce773a501441d204aa9e",
          "commission": "10",
          "created_at_block": "9",
          "status": 1
        }
      ]
    }

Coin Info
^^^^^^^^^

Returns information about coin.

*Note*: this method **does not** return information about base coins (MNT and BIP).

.. code-block:: bash

    curl -s 'localhost:8841/coin_info?symbol={SYMBOL}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "name": "TEST",
        "symbol": "TESTCOIN",
        "volume": "100000000000000000000",
        "crr": "100",
        "reserve_balance": "100000000000000000000"
      }
    }

**Result**:
    - **Coin name** - Name of a coin. Arbitrary string.
    - **Coin symbol** - Short symbol of a coin. Coin symbol is unique, alphabetic, uppercase, 3 to 10 letters length.
    - **Volume** - Amount of coins exists in network.
    - **Reserve balance** - Amount of BIP/MNT in coin reserve.
    - **Constant Reserve Ratio (CRR)** - uint, from 10 to 100.
    - **Creator** - Address of coin creator account.

Estimate sell coin
^^^^^^^^^^^^^^^^^^

Return estimate of sell coin transaction

.. code-block:: bash

    curl -s 'localhost:8841/estimate_coin_sell?coin_to_sell=MNT&coin_to_buy=TESTCOIN&value_to_sell=1'

Request params:
    - **coin_to_sell** – coin to give
    - **value_to_sell** – amount to give (in pips)
    - **coin_to_buy** - coin to get

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "will_pay": "1",
        "commission": "100000000000000000"
      }
    }

**Result**: Amount of "to_coin" user should get.

Estimate buy coin
^^^^^^^^^^^^^^^^^

Return estimate of buy coin transaction

.. code-block:: bash

    curl -s 'localhost:8841/estimate_coin_buy?coin_to_sell=MNT&coin_to_buy=TESTCOIN&value_to_buy=1'

Request params:
    - **coin_to_sell** – coin to give
    - **value_to_buy** – amount to get (in pips)
    - **coin_to_buy** - coin to get

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "will_pay": "1",
        "commission": "100000000000000000"
      }
    }


**Result**: Amount of "to_coin" user should give.

Estimate tx commission
^^^^^^^^^^^^^^^^^^^^^^

Return estimate of buy coin transaction

.. code-block:: bash

    curl -s 'localhost:8841/estimate_tx_commission?tx={transaction}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "commission": "11000000000000000000"
      }
    }

**Result**: Commission in GasCoin.
