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
            "latest_block_hash": "30AAD93FC07CBFC7ABC9E34D6FDC29FF0928A5C5",
            "latest_app_hash": "8D10D20C2BC74AAF82ABC41ADA9852D5EF89DDE17382CED2C21B84BE36365583",
            "latest_block_height": 29783,
            "latest_block_time": "2018-06-21T13:58:53.078510484+03:00"
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
        "code": 0,
        "result": "20000111000000000000000000"
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
                "candidate_address": "Mx655a96de0e7928bf78c41f555010391581a5afab",
                "total_stake": "49500000000000000000",
                "pub_key": "Mp34e647f46a5dd89e9f21acdbd0c45c8c768fdc17082d0783b683bfb0da7ce989",
                "commission": 50,
                "accumulated_reward": "0",
                "stakes": [
                    {
                        "owner": "Mx655a96de0e7928bf78c41f555010391581a5afab",
                        "coin": "MNT",
                        "value": "49500000000000000000"
                    }
                ],
                "created_at_block": 27447,
                "status": 1,
                "absent_times": 0
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
                "candidate_address": "Mx655a96de0e7928bf78c41f555010391581a5afab",
                "total_stake": "49500000000000000000",
                "pub_key": "Mp34e647f46a5dd89e9f21acdbd0c45c8c768fdc17082d0783b683bfb0da7ce989",
                "commission": 50,
                "accumulated_reward": "0",
                "stakes": [
                    {
                        "owner": "Mx655a96de0e7928bf78c41f555010391581a5afab",
                        "coin": "MNT",
                        "value": "49500000000000000000"
                    }
                ],
                "created_at_block": 27447,
                "status": 1,
                "absent_times": 0
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
            "MNT": "670983232356790123336"
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
        "result": 3
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
        "result": "Mtfd5c3ecad1e8333564cf6e3f968578b9db5acea3"
    }

**Result**: Transaction hash.

Transaction
^^^^^^^^^^^

*In development*

.. code-block:: bash

    curl -s 'localhost:8841/api/transaction/{hash}'

.. code-block:: json

    {
        "code": 0,
        "result": {}
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
          "hash":"A83F3A3909C8B863305C5A444C8C34C514A03590",
          "height":108805,
          "time":"2018-07-03T09:46:54.359423195Z",
          "num_txs":1,
          "total_txs":1174135,
          "transactions":[
             {
                "hash":"Mt3f85c77911f058c9c2f79d73c5d68b2c7dd3c2cd",
                "from":"Mxa93163fdF10724DC4785FF5cBfB9aC0B5949409F",
                "nonce":81,
                "gasPrice":1,
                "type":5,
                "data":{
                   "PubKey":"Mp079138d379aaf423c911506a3ccbe1d590a7d4d9aecbc7eb05816d81b41848d6",
                   "Coin":"BLTCOIN",
                   "Stake":"2000000000000000000"
                },
                "payload":"",
                "serviceData":"",
                "gas":10000
             }
          ],
          "precommits":[
             {
                "validator_address":"04E5DCA0DFCF35605A3EB1292DBDBF7C97B476B8",
                "validator_index":0,
                "height":108804,
                "round":0,
                "timestamp":"2018-07-03T09:47:33.79209988Z",
                "type":2,
                "block_id":{
                   "hash":"2222959DA3EEA441DB6D0E01C12F1546B210DA72",
                   "parts":{
                      "total":1,
                      "hash":"3821D8B2A09A1C6932712523B8DEB588375D7BFA"
                   }
                },
                "signature":[]
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

Exchange estimate
^^^^^^^^^^^^^^^^^

Return estimate of coin exchange transaction

.. code-block:: bash

    curl -s 'localhost:8841/api/estimateCoinExchangeReturn?from_coin=MNT&value=1000000000000000000&to_coin=BLTCOIN'

Request params:
    - **from_coin** – coin to give
    - **value** – amount to give (in pips)
    - **to_coin** - coin to get

.. code-block:: json

    {
        "code": 0,
        "result": "29808848728151191"
    }

**Result**: Amount of "to_coin" user will receive.
