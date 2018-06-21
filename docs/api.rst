Minter Node API
===============

Status
^^^^^^

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

.. code-block:: bash

    curl -s 'localhost:8841/api/bipVolume?height={height}'

.. code-block:: json

    {
        "code": 0,
        "result": "20000111000000000000000000"
    }

Candidate
^^^^^^^^^

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

.. code-block:: bash

    curl -s 'localhost:8841/api/balance/{address}'

.. code-block:: json

    {
        "code": 0,
        "result": {
            "MNT": "670983232356790123336"
        }
    }

Transaction count
^^^^^^^^^^^^^^^^^

.. code-block:: bash

    curl -s 'localhost:8841/api/transactionCount/{address}'

.. code-block:: json

    {
        "code": 0,
        "result": 3
    }

Send transaction
^^^^^^^^^^^^^^^^

.. code-block:: bash

    curl -X POST --data '{"transaction":"..."}' -s 'localhost:8841/api/sendTransaction'

.. code-block:: json

    {
        "code": 0,
        "result": "Mtfd5c3ecad1e8333564cf6e3f968578b9db5acea3"
    }

Transaction
^^^^^^^^^^^

.. code-block:: bash

    curl -s 'localhost:8841/api/transaction/{hash}'

.. code-block:: json

    {
        "code": 0,
        "result": ...
    }

Block
^^^^^

.. code-block:: bash

    curl -s 'localhost:8841/api/block/{height}'

.. code-block:: json

    {
        "code": 0,
        "result": ...
    }

Coin Info
^^^^^^^^^

.. code-block:: bash

    curl -s 'localhost:8841/api/coinInfo/{symbol}'

.. code-block:: json

    {
        "code": 0,
        "result": ...
    }
