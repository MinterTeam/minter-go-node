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
      "code": 0,
      "result": {
        "version": "0.2.5",
        "latest_block_hash": "0CC015EA926173130C793BBE6E38145BF379CF6A",
        "latest_app_hash": "1FB9B53F32298759D936E4A10A866E7AFB930EA4D7CC7184EC992F2320592E81",
        "latest_block_height": 82541,
        "latest_block_time": "2018-08-28T18:26:47.112704193+03:00",
        "tm_status": {
          "node_info": {
            "id": "62a5d75ef3f48dcf62aad263a170b9c82eb3f2b8",
            "listen_addr": "192.168.1.100:26656",
            "network": "minter-test-network-19",
            "version": "0.23.0",
            "channels": "4020212223303800",
            "moniker": "MinterNode",
            "other": [
              "amino_version=0.10.1",
              "p2p_version=0.5.0",
              "consensus_version=v1/0.2.2",
              "rpc_version=0.7.0/3",
              "tx_index=on",
              "rpc_addr=tcp://0.0.0.0:26657"
            ]
          },
          "sync_info": {
            "latest_block_hash": "0CC015EA926173130C793BBE6E38145BF379CF6A",
            "latest_app_hash": "1FB9B53F32298759D936E4A10A866E7AFB930EA4D7CC7184EC992F2320592E81",
            "latest_block_height": "82541",
            "latest_block_time": "2018-08-28T15:26:47.112704193Z",
            "catching_up": true
          },
          "validator_info": {
            "address": "BCFB297FD1EE0458E1DBDA8EBAE2C599CD0A5984",
            "pub_key": {
              "type": "tendermint/PubKeyEd25519",
              "value": "G2lZ+lJWW/kQvhOOI6CHVBHSEgjYq9awDgdlErLeVAE="
            },
            "voting_power": "0"
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
      "code": 0,
      "result": {
        "candidate": {
          "candidate_address": "Mxee81347211c72524338f9680072af90744333146",
          "total_stake": "5000001000000000000000000",
          "pub_key": "Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd1c",
          "commission": 100,
          "stakes": [
            {
              "owner": "Mxee81347211c72524338f9680072af90744333146",
              "coin": "MNT",
              "value": "5000000000000000000000000",
              "bip_value": "5000000000000000000000000"
            },
            {
              "owner": "Mx4f3385615a4abb104d6eda88591fa07c112cbdbf",
              "coin": "MNT",
              "value": "1000000000000000000",
              "bip_value": "1000000000000000000"
            }
          ],
          "created_at_block": 165,
          "status": 2
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
      "code": 0,
      "result": [
        {
          "accumulated_reward": "652930049792069211272",
          "absent_times": 0,
          "candidate": {
            "candidate_address": "Mxee81347211c72524338f9680072af90744333146",
            "total_stake": "5000001000000000000000000",
            "pub_key": "Mp738da41ba6a7b7d69b7294afa158b89c5a1b410cbf0c2443c85c5fe24ad1dd1c",
            "commission": 100,
            "stakes": [
              {
                "owner": "Mxee81347211c72524338f9680072af90744333146",
                "coin": "MNT",
                "value": "5000000000000000000000000",
                "bip_value": "5000000000000000000000000"
              },
              {
                "owner": "Mx4f3385615a4abb104d6eda88591fa07c112cbdbf",
                "coin": "MNT",
                "value": "1000000000000000000",
                "bip_value": "1000000000000000000"
              }
            ],
            "created_at_block": 165,
            "status": 2
          }
        },
        {
          "accumulated_reward": "652929919206085370058",
          "absent_times": 0,
          "candidate": {
            "candidate_address": "Mxee81347211c72524338f9680072af90744333146",
            "total_stake": "5000000000000000000000000",
            "pub_key": "Mp6f16c1ff21a6fb946aaed0f4c1fcca272b72fd904988f91d3883282b8ae31ba2",
            "commission": 100,
            "stakes": [
              {
                "owner": "Mxee81347211c72524338f9680072af90744333146",
                "coin": "MNT",
                "value": "5000000000000000000000000",
                "bip_value": "5000000000000000000000000"
              }
            ],
            "created_at_block": 174,
            "status": 2
          }
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
      "code": 0,
      "result": {
        "balance": {
          "MINTERONE": "2000000000000000000",
          "MNT": "97924621949581028367025445",
          "SHSCOIN": "201502537939970000000000",
          "TESTCOIN": "1000000000000000000000"
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
      "code": 0,
      "result": {
        "count": 59
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
      "code": 0,
      "result": {
        "hash": "E9BC108B9C9B3D9BC276EE359BF9DD98C144B7C6",
        "raw_tx": "f8818207af018a4d4e540000000000000001abea8a4d4e54000000000000009435d05ae08a664964ba730ca7e7de6e97998086f589056bc75e2d6310000080801ba076d4aeb96756d94db0ad0fdb73aaff588f4df282b64b4dec34930dba3ca2ffc5a04e47954ec056235103707c5aeb33a9112eab6d63de6b9a3d9e7a156c3bebeca7",
        "height": 94594,
        "index": 0,
        "tx_result": {
          "gas_wanted": 10,
          "gas_used": 10,
          "tags": [
            {
              "key": "dHgudHlwZQ==",
              "value": "AQ=="
            },
            {
              "key": "dHguZnJvbQ==",
              "value": "ZmU2MDAxNGE2ZTlhYzkxNjE4ZjVkMWNhYjNmZDU4Y2RlZDYxZWU5OQ=="
            },
            {
              "key": "dHgudG8=",
              "value": "MzVkMDVhZTA4YTY2NDk2NGJhNzMwY2E3ZTdkZTZlOTc5OTgwODZmNQ=="
            },
            {
              "key": "dHguY29pbg==",
              "value": "TU5U"
            }
          ]
        },
        "from": "Mxfe60014a6e9ac91618f5d1cab3fd58cded61ee99",
        "nonce": 1967,
        "gas_price": 1,
        "gas_coin": "MNT",
        "type": 1,
        "data": {
          "coin": "MNT",
          "to": "Mx35d05ae08a664964ba730ca7e7de6e97998086f5",
          "value": "100000000000000000000"
        },
        "payload": ""
      }
    }

Block
^^^^^

Returns block data at given height.

.. code-block:: bash

    curl -s 'localhost:8841/api/block/{height}'

.. code-block:: json

    {
      "code": 0,
      "result": {
        "hash": "6B4F84E0C801EE01B4EA1AEC34B0A0249E4EB3FF",
        "height": 94594,
        "time": "2018-08-29T10:12:52.791097555Z",
        "num_txs": 1,
        "total_txs": 5515,
        "transactions": [
          {
            "hash": "Mte9bc108b9c9b3d9bc276ee359bf9dd98c144b7c6",
            "raw_tx": "f8818207af018a4d4e540000000000000001abea8a4d4e54000000000000009435d05ae08a664964ba730ca7e7de6e97998086f589056bc75e2d6310000080801ba076d4aeb96756d94db0ad0fdb73aaff588f4df282b64b4dec34930dba3ca2ffc5a04e47954ec056235103707c5aeb33a9112eab6d63de6b9a3d9e7a156c3bebeca7",
            "from": "Mxfe60014a6e9ac91618f5d1cab3fd58cded61ee99",
            "nonce": 1967,
            "gas_price": 1,
            "type": 1,
            "data": {
              "coin": "MNT",
              "to": "Mx35d05ae08a664964ba730ca7e7de6e97998086f5",
              "value": "100000000000000000000"
            },
            "payload": "",
            "service_data": "",
            "gas": 10,
            "gas_coin": "MNT",
            "tx_result": {
              "gas_wanted": 10,
              "gas_used": 10,
              "tags": [
                {
                  "key": "dHgudHlwZQ==",
                  "value": "AQ=="
                },
                {
                  "key": "dHguZnJvbQ==",
                  "value": "ZmU2MDAxNGE2ZTlhYzkxNjE4ZjVkMWNhYjNmZDU4Y2RlZDYxZWU5OQ=="
                },
                {
                  "key": "dHgudG8=",
                  "value": "MzVkMDVhZTA4YTY2NDk2NGJhNzMwY2E3ZTdkZTZlOTc5OTgwODZmNQ=="
                },
                {
                  "key": "dHguY29pbg==",
                  "value": "TU5U"
                }
              ]
            }
          }
        ],
        "precommits": [
          {
            "validator_address": "0D1A38E170F4BC84CBA505E041AF0A656FEF7CCE",
            "validator_index": "0",
            "height": "94593",
            "round": "0",
            "timestamp": "2018-08-29T10:12:47.480971248Z",
            "type": 2,
            "block_id": {
              "hash": "CCC196AE488111387594258B4F5B417B6DF6F01E",
              "parts": {
                "total": "1",
                "hash": "6C8A070EBDD7218547617CD2E0894E031B815B95"
              }
            },
            "signature": "+tNZnoPJnQNpanlK90YEb11GnGP20wGzrrqX7Wzf729KhZBhOkK4zFZW0CnUfVHwYpu4nGVaJLOgy8G6VKCgCg=="
          },
          {
            "validator_address": "1B16468F89B8C36FE1AFC7F82F7251D4FC831530",
            "validator_index": "1",
            "height": "94593",
            "round": "0",
            "timestamp": "2018-08-29T10:12:47.494792759Z",
            "type": 2,
            "block_id": {
              "hash": "CCC196AE488111387594258B4F5B417B6DF6F01E",
              "parts": {
                "total": "1",
                "hash": "6C8A070EBDD7218547617CD2E0894E031B815B95"
              }
            },
            "signature": "GofqbrNFZye3pQk8sDsuErFH4x4Z+bs7skQOeeTcNA+jSIoupo+NWM6SV/rePg6NVOSA3PHVkXG6MVO2xYfbCg=="
          },
          {
            "validator_address": "22794FF373BE0867ECCB8206BEB77E0AB6F4A198",
            "validator_index": "2",
            "height": "94593",
            "round": "0",
            "timestamp": "2018-08-29T10:12:47.465617407Z",
            "type": 2,
            "block_id": {
              "hash": "CCC196AE488111387594258B4F5B417B6DF6F01E",
              "parts": {
                "total": "1",
                "hash": "6C8A070EBDD7218547617CD2E0894E031B815B95"
              }
            },
            "signature": "19Xu5Y8UI4QwZc89HgC42G4dB8MaMn7ibph6R1iVo9YYwwTKN4NEOjbuvvl3VYl8k/8CBIhck45GtSq73xHiBA=="
          },
          {
            "validator_address": "36575649BE18934623E0CE226B8E60FB1D1E7163",
            "validator_index": "3",
            "height": "94593",
            "round": "0",
            "timestamp": "2018-08-29T10:12:47.488838407Z",
            "type": 2,
            "block_id": {
              "hash": "CCC196AE488111387594258B4F5B417B6DF6F01E",
              "parts": {
                "total": "1",
                "hash": "6C8A070EBDD7218547617CD2E0894E031B815B95"
              }
            },
            "signature": "5PE9BYgsnXGtzUeeUBqIwA/VTfunHC+gN1keQYeN220JSjXrI7qZguYm45+9dt79s/y6jc8S4XKRSDNvVI1DDg=="
          },
          {
            "validator_address": "6330D572B9670786E0603332C01E7D4C35653C4A",
            "validator_index": "4",
            "height": "94593",
            "round": "0",
            "timestamp": "2018-08-29T10:12:47.443544695Z",
            "type": 2,
            "block_id": {
              "hash": "CCC196AE488111387594258B4F5B417B6DF6F01E",
              "parts": {
                "total": "1",
                "hash": "6C8A070EBDD7218547617CD2E0894E031B815B95"
              }
            },
            "signature": "QTW+t2Yen2U04gO3T3CRG3nhAkmkVM1ucQRD4QZS5Pokwp8C9ykP3uEefXjgTznBd3x24+hkTHSUfOy/HY9CCw=="
          }
        ],
        "block_reward": "333000000000000000000"
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
      "code": 0,
      "result": {
        "name": "Stakeholder Coin",
        "symbol": "SHSCOIN",
        "volume": "1985888114702108355026636",
        "crr": 50,
        "reserve_balance": "394375160721239016660255",
        "creator": "Mx6eadf5badeda8f76fc35e0c4d7f7fbc00fe34315"
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

Estimate tx commission
^^^^^^^^^^^^^^^^^^^^^^

Return estimate of buy coin transaction

.. code-block:: bash

    curl -s 'localhost:8841/api/estimateTxCommission?tx={transaction}'

.. code-block:: json

    {
      "code": 0,
      "result": {
        "commission": "10000000000000000"
      }
    }


**Result**: Commission in GasCoin.
