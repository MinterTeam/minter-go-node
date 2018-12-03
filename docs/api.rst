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



**Result->balance**: Map of balances. CoinSymbol => Balance (in pips).
**Result->transaction_count**: Count of transactions sent from the account.

Send transaction
^^^^^^^^^^^^^^^^

Sends transaction to the Minter Network.

.. code-block:: bash

    curl -s 'localhost:8841/send_transaction?tx={transaction}'

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

    curl -s 'localhost:8841/transaction?hash={hash}'

.. code-block:: json

    {
        "code": 0,
        "result": {
            "hash": "B829EE45734800273ACCCFA70BC96BE8D858E521",
            "raw_tx": "f88682c8e8018a4d4e540000000000000001abea8a4d4e5400000000000000941a8e2cd08a2938b6412cc65aed449154577731e089056bc75e2d63100000808001b845f8431ba050141c66539362464496d1393b8a9468623f37dced4cf3bac8bfc5d576fd5e1fa00b37ee7103d2647bd7fc1a72c03df2a40cc38c9cc92771614f76eabf7be1cc79",
            "height": 234218,
            "index": 0,
            "from": "Mxfe60014a6e9ac91618f5d1cab3fd58cded61ee99",
            "nonce": 51432,
            "gas_price": 1,
            "gas_coin": "MNT",
            "gas_used": 10,
            "type": 1,
            "data": {
                "coin": "MNT",
                "to": "Mx1a8e2cd08a2938b6412cc65aed449154577731e0",
                "value": "100000000000000000000"
            },
            "payload": "",
            "tags": {
                "tx.coin": "MNT",
                "tx.from": "fe60014a6e9ac91618f5d1cab3fd58cded61ee99",
                "tx.to": "1a8e2cd08a2938b6412cc65aed449154577731e0",
                "tx.type": "01"
            }
        }
    }

Block
^^^^^

Returns block data at given height.

.. code-block:: bash

    curl -s 'localhost:8841/block/{height}'

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
            "hash": "B829EE45734800273ACCCFA70BC96BE8D858E521",
            "raw_tx": "f88682c8e8018a4d4e540000000000000001abea8a4d4e5400000000000000941a8e2cd08a2938b6412cc65aed449154577731e089056bc75e2d63100000808001b845f8431ba050141c66539362464496d1393b8a9468623f37dced4cf3bac8bfc5d576fd5e1fa00b37ee7103d2647bd7fc1a72c03df2a40cc38c9cc92771614f76eabf7be1cc79",
            "height": 94594,
            "index": 0,
            "from": "Mxfe60014a6e9ac91618f5d1cab3fd58cded61ee99",
            "nonce": 51432,
            "gas_price": 1,
            "gas_coin": "MNT",
            "gas_used": 10,
            "type": 1,
            "data": {
                "coin": "MNT",
                "to": "Mx1a8e2cd08a2938b6412cc65aed449154577731e0",
                "value": "100000000000000000000"
            },
            "payload": "",
            "tags": {
                "tx.coin": "MNT",
                "tx.from": "fe60014a6e9ac91618f5d1cab3fd58cded61ee99",
                "tx.to": "1a8e2cd08a2938b6412cc65aed449154577731e0",
                "tx.type": "01"
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

    curl -s 'localhost:8841/coinInfo/{symbol}'

.. code-block:: json

    {
      "code": 0,
      "result": {
        "name": "Stakeholder Coin",
        "symbol": "SHSCOIN",
        "volume": "1985888114702108355026636",
        "crr": 50,
        "reserve_balance": "394375160721239016660255"
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

    curl -s 'localhost:8841/estimateCoinSell?coin_to_sell=MNT&value_to_sell=1000000000000000000&coin_to_buy=BLTCOIN'

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

    curl -s 'localhost:8841/estimateCoinBuy?coin_to_sell=MNT&value_to_buy=1000000000000000000&coin_to_buy=BLTCOIN'

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

    curl -s 'localhost:8841/estimateTxCommission?tx={transaction}'

.. code-block:: json

    {
      "code": 0,
      "result": {
        "commission": "10000000000000000"
      }
    }


**Result**: Commission in GasCoin.
