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

    curl -s 'localhost:8841/api/status'

.. code-block:: json

    {
       "code":0,
       "result":{
          "version":"0.1.0",
          "latest_block_hash":"BF2647887AEBF12ABF92D240613907E84E757E34",
          "latest_app_hash":"C92D2073E15519C0D684A896AF8DF9AAD536423A9564987F979CFCC13FBE57D7",
          "latest_block_height":81,
          "latest_block_time":"2018-07-20T16:03:42.001313931+03:00",
          "tm_status":{
             "node_info":{
                "id":"30231c71e87db942ea902ad6ad22cfefa3b15560",
                "listen_addr":"192.168.1.102:26656",
                "network":"minter-test-network-11-private",
                "version":"0.22.4",
                "channels":"4020212223303800",
                "moniker":"MinterNode",
                "other":[
                   "amino_version=0.10.1",
                   "p2p_version=0.5.0",
                   "consensus_version=v1/0.2.2",
                   "rpc_version=0.7.0/3",
                   "tx_index=on",
                   "rpc_addr=tcp://0.0.0.0:26657"
                ]
             },
             "sync_info":{
                "latest_block_hash":"BF2647887AEBF12ABF92D240613907E84E757E34",
                "latest_app_hash":"C92D2073E15519C0D684A896AF8DF9AAD536423A9564987F979CFCC13FBE57D7",
                "latest_block_height":"81",
                "latest_block_time":"2018-07-20T13:03:42.001313931Z",
                "catching_up":false
             },
             "validator_info":{
                "address":"F974AA1C211BC294DAB21B4F5866603144E025E8",
                "pub_key":{
                   "type":"tendermint/PubKeyEd25519",
                   "value":"YfdhnC3qkBZqgQl76+lY99f0xfGJLyTdgTOLJ2CSvnA="
                },
                "voting_power":"0"
             }
          }
       }
    }

Volume of Base Coin in Blockchain
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

This endpoint shows amount of base coin (BIP or MNT) existing in the network. It counts block rewards, premine and
relayed rewards.

.. code-block:: bash

    curl -s 'localhost:8841/api/bipVolume?height={height}'

.. code-block:: json

    {
       "code":0,
       "result":{
          "volume":"20000222000000000000000000"
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

    curl -s 'localhost:8841/api/candidate/{public_key}'

.. code-block:: json

    {
       "code":0,
       "result":{
          "candidate":{
             "candidate_address":"Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f",
             "total_stake":"1",
             "pub_key":"Mpc0d436ce0a9e7129cb3dbbfb059ec3a45865305a4102bc68cf6ed41d41d53e99",
             "commission":10,
             "accumulated_reward":"0",
             "stakes":[
                {
                   "owner":"Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f",
                   "coin":"MNT",
                   "value":"1"
                }
             ],
             "created_at_block":1,
             "status":2,
             "absent_times":0
          }
       }
    }

Validators
^^^^^^^^^^

Returns list of active validators.

.. code-block:: bash

    curl -s 'localhost:8841/api/validators'

.. code-block:: json

    {
       "code":0,
       "result":[
          {
             "candidate_address":"Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f",
             "total_stake":"1",
             "pub_key":"Mpc0d436ce0a9e7129cb3dbbfb059ec3a45865305a4102bc68cf6ed41d41d53e99",
             "commission":10,
             "accumulated_reward":"666000000000000000000",
             "stakes":[
                {
                   "owner":"Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f",
                   "coin":"MNT",
                   "value":"1"
                }
             ],
             "created_at_block":1,
             "status":2,
             "absent_times":0
          }
       ]
    }

Balance
^^^^^^^

Returns balance of an account.

.. code-block:: bash

    curl -s 'localhost:8841/api/balance/{address}'

.. code-block:: json

    {
       "code":0,
       "result":{
          "balance":{
             "MNT":"100011877000000000000000000"
          }
       }
    }

**Result**: Map of balances. CoinSymbol => Balance (in pips).

Transaction count
^^^^^^^^^^^^^^^^^

Returns count of outgoing transactions from given account. This should be used for calculating nonce for the new
transaction.

.. code-block:: bash

    curl -s 'localhost:8841/api/transactionCount/{address}'

.. code-block:: json

    {
       "code":0,
       "result":{
          "count":1
       }
    }

**Result**: Count of transactions sent from given account.

Send transaction
^^^^^^^^^^^^^^^^

Sends transaction to the Minter Network.

.. code-block:: bash

    curl -X POST --data '{"transaction":"..."}' -s 'localhost:8841/api/sendTransaction'

.. code-block:: json

    {
        "code": 0,
        "result": {
            "hash": "Mtfd5c3ecad1e8333564cf6e3f968578b9db5acea3"
        }
    }

**Result**: Transaction hash.

Transaction
^^^^^^^^^^^

.. code-block:: bash

    curl -s 'localhost:8841/api/transaction/{hash}'

.. code-block:: json

    {
       "code":0,
       "result":{
          "hash":"47B0CF9BFAA60CA343392FBE1E366EB221231F38",
          "raw_tx":"f873010101aae98a4d4e540000000000000094a93163fdf10724dc4785ff5cbfb9ac0b5949409f880de0b6b3a764000080801ba0da1b6fd187bc5c757d1d1497d03471a3b5d1fd4d8025859ea127841975ce0df4a0158b54aaf8066be9ef26aae9f1a953777c346e58a6c6f45eb2d465efea74e5af",
          "height":41,
          "index":0,
          "tx_result":{
             "gas_wanted":10,
             "gas_used":10,
             "tags":[
                {
                   "key":"dHgudHlwZQ==",
                   "value":"AQ=="
                },
                {
                   "key":"dHguZnJvbQ==",
                   "value":"YTkzMTYzZmRmMTA3MjRkYzQ3ODVmZjVjYmZiOWFjMGI1OTQ5NDA5Zg=="
                },
                {
                   "key":"dHgudG8=",
                   "value":"YTkzMTYzZmRmMTA3MjRkYzQ3ODVmZjVjYmZiOWFjMGI1OTQ5NDA5Zg=="
                },
                {
                   "key":"dHguY29pbg==",
                   "value":"TU5U"
                }
             ],
             "fee":{

             }
          },
          "from":"Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f",
          "nonce":1,
          "gas_price":1,
          "type":1,
          "data":{
             "coin":"MNT",
             "to":"Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f",
             "value":"1000000000000000000"
          },
          "payload":""
       }
    }

Block
^^^^^

Returns block data at given height.

.. code-block:: bash

    curl -s 'localhost:8841/api/block/{height}'

.. code-block:: json

    {
       "code":0,
       "result":{
          "hash":"8E07E206FBB41D7697D105CBC7FE477DDFAA2D5B",
          "height":41,
          "time":"2018-07-20T13:00:21.575014435Z",
          "num_txs":1,
          "total_txs":1,
          "transactions":[
             {
                "hash":"Mt47b0cf9bfaa60ca343392fbe1e366eb221231f38",
                "raw_tx":"f873010101aae98a4d4e540000000000000094a93163fdf10724dc4785ff5cbfb9ac0b5949409f880de0b6b3a764000080801ba0da1b6fd187bc5c757d1d1497d03471a3b5d1fd4d8025859ea127841975ce0df4a0158b54aaf8066be9ef26aae9f1a953777c346e58a6c6f45eb2d465efea74e5af",
                "from":"Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f",
                "nonce":1,
                "gas_price":1,
                "type":1,
                "data":{
                   "coin":"MNT",
                   "to":"Mxa93163fdf10724dc4785ff5cbfb9ac0b5949409f",
                   "value":"1000000000000000000"
                },
                "payload":"",
                "service_data":"",
                "gas":10,
                "tx_result":{
                   "gas_wanted":10,
                   "gas_used":10,
                   "tags":[
                      {
                         "key":"dHgudHlwZQ==",
                         "value":"AQ=="
                      },
                      {
                         "key":"dHguZnJvbQ==",
                         "value":"YTkzMTYzZmRmMTA3MjRkYzQ3ODVmZjVjYmZiOWFjMGI1OTQ5NDA5Zg=="
                      },
                      {
                         "key":"dHgudG8=",
                         "value":"YTkzMTYzZmRmMTA3MjRkYzQ3ODVmZjVjYmZiOWFjMGI1OTQ5NDA5Zg=="
                      },
                      {
                         "key":"dHguY29pbg==",
                         "value":"TU5U"
                      }
                   ],
                   "fee":{

                   }
                }
             }
          ],
          "precommits":[
             {
                "validator_address":"8055BB821C535279E169FDF60BBEBEBE1452DBA8",
                "validator_index":"0",
                "height":"40",
                "round":"0",
                "timestamp":"2018-07-20T13:00:16.571443571Z",
                "type":2,
                "block_id":{
                   "hash":"0F06CA442183BED91E66010314FA6CADBC598801",
                   "parts":{
                      "total":"1",
                      "hash":"3D119516E329A211B74D728728A7E283E3BC956E"
                   }
                },
                "signature":{
                   "type":"tendermint/SignatureEd25519",
                   "value":"lhNyaFgSYmC7YF/FPSwZ2yksWwViaclK6rGwdN2+nVnp/uMQherRMyZv6hJB/YedAjgo49/fBhGZUcyOO7Y+AA=="
                }
             }
          ]
       }
    }

Coin Info
^^^^^^^^^

Returns information about coin.

*Note*: this method **does not** return information about base coins (MNT and BIP).

.. code-block:: bash

    curl -s 'localhost:8841/api/coinInfo/{symbol}'

.. code-block:: json

    {
       "code":0,
       "result":{
          "name":"BeltCoin",
          "symbol":"BLTCOIN",
          "volume":"3162375676992609621",
          "crr":10,
          "reserve_balance":"100030999965000000000000",
          "creator":"Mxc07ec7cdcae90dea3999558f022aeb25dabbeea2"
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

    curl -s 'localhost:8841/api/estimateCoinSell?coin_to_sell=MNT&value_to_sell=1000000000000000000&coin_to_buy=BLTCOIN'

Request params:
    - **coin_to_sell** – coin to give
    - **value_to_sell** – amount to give (in pips)
    - **coin_to_buy** - coin to get

.. code-block:: json

    {
        "code": 0,
        "result": {
            "will_get": "29808848728151191",
            "commission": "443372813245"
        }
    }

**Result**: Amount of "to_coin" user should get.


Estimate buy coin
^^^^^^^^^^^^^^^^^

Return estimate of buy coin transaction

.. code-block:: bash

    curl -s 'localhost:8841/api/estimateCoinBuy?coin_to_sell=MNT&value_to_buy=1000000000000000000&coin_to_buy=BLTCOIN'

Request params:
    - **coin_to_sell** – coin to give
    - **value_to_buy** – amount to get (in pips)
    - **coin_to_buy** - coin to get

.. code-block:: json

    {
        "code": 0,
        "result": {
            "will_pay": "29808848728151191",
            "commission": "443372813245"
        }
    }

**Result**: Amount of "to_coin" user should give.
