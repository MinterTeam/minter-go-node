Minter Node API
===============

Minter Node API is based on JSON format. JSON is a lightweight data-interchange format.
It can represent numbers, strings, ordered sequences of values, and collections of name/value pairs.

If request is successful, Minter Node API will respond with ``result`` key and code equal to zero. Otherwise, it will
respond with non-zero code and key ``log`` with error description.

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
        "candidate_address": "Mxee81347211c72524338f9680072af90744333146",
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
          "accumulated_reward": 2331000000000000000000,
          "absent_times": "0",
          "candidate": {
            "candidate_address": "Mxee81347211c72524338f9680072af90744333146",
            "total_stake": 0,
            "pub_key": "Mpe0ba50e3468b07fbbc127840953eb8f4fe57d6e8162df93baefd79f5d5bc2b97",
            "commission": "100",
            "created_at_block": "1",
            "status": 2
          }
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

Sends transaction to the Minter Network.

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
        "hash": "F4D2F2DF68B20275B832B2D1859308509C373523689259CABD17AFC777C0B014",
        "height": "387",
        "time": "2018-12-03T14:39:41.364276Z",
        "num_txs": "1",
        "total_txs": "1",
        "transactions": [
          {
            "hash": "Mtc6c6b5008af8077fb0ce817ddb79268d1c66b6b353af76778ca5a264a80069db",
            "raw_tx": "f88701018a4d4e540000000000000001aae98a4d4e540000000000000094ee81347211c72524338f9680072af9074433314688a688906bd8b0000084546573748001b845f8431ba098fd9402b0af434f461eecdad89908655c779fb394b7624a0c37198f931f27a1a075e73a04f81e2204d88826ac851b2b3da359e4a9a16ac6c17e992fa0a3de0c48",
            "from": "Mxee81347211c72524338f9680072af90744333146",
            "nonce": "1",
            "gas_price": "1",
            "type": 1,
            "data": {
              "coin": "MNT",
              "to": "Mxee81347211c72524338f9680072af90744333146",
              "value": "12000000000000000000"
            },
            "payload": "VGVzdA==",
            "service_data": "",
            "gas": "18",
            "gas_coin": "MNT",
            "gas_used": "18",
            "tags": {
              "tx.type": "01",
              "tx.from": "ee81347211c72524338f9680072af90744333146",
              "tx.to": "ee81347211c72524338f9680072af90744333146",
              "tx.coin": "MNT"
            }
          }
        ],
        "precommits": [
          {
            "type": 2,
            "height": "386",
            "round": "0",
            "timestamp": "2018-12-03T14:39:41.364276Z",
            "block_id": {
              "hash": "8348B85D729555F0FEC3258EE07B188A38702F5045C1C0E3F0200A59713AA32F",
              "parts": {
                "total": "1",
                "hash": "5816E926E9006AF09D6E77A51A90051B54E2D6FF984A21BE4AAC39A0E0758678"
              }
            },
            "validator_address": "AB15A084DD592699812E9B22385C1959E7AEFFB8",
            "validator_index": "0",
            "signature": "V/9uM9ZVUgfh5NdZn0DS4xWubLqJiEAB+J1McBCKW0Kq2X25L4+6mSjEG/hSYNxLGtS+yV22bhxLcc2qvqoDAg=="
          }
        ],
        "block_reward": "333000000000000000000",
        "size": "1204"
      }
    }



Coin Info
^^^^^^^^^

Returns information about coin.

*Note*: this method **does not** return information about base coins (MNT and BIP).

.. code-block:: bash

    curl -s 'localhost:8841/coin_info?symbol="{SYMBOL}"'

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

    curl -s 'localhost:8841/estimate_coin_sell?coin_to_sell="MNT"&coin_to_buy="TESTCOIN"&value_to_sell="1"'

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

    curl -s 'localhost:8841/estimate_coin_buy?coin_to_sell="MNT"&coin_to_buy="TESTCOIN"&value_to_buy="1"'

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

    curl -s 'localhost:8841/estimateTxCommission?tx={transaction}'

.. code-block:: json

    {
      "jsonrpc": "2.0",
      "id": "",
      "result": {
        "commission": "11000000000000000000"
      }
    }



**Result**: Commission in GasCoin.
