Transactions
============

Semantic
^^^^^^^^

Transactions in Minter are `LRP-encoded <https://github.com/ethereum/wiki/wiki/RLP>`__ structures.

Example of a signed transaction:

::

    f873230101aae98a4d4e540000000000000094a93163fdf10724dc4785ff5cbfb9
    ac0b5949409f880de0b6b3a764000080801ba06838db4a2197cfd70ede8d8d184d
    bf332932ca051a243eb7886791250e545dd3a04b63fb1d1b5ef5f2cbd2ea12530c
    da520b3280dcb75bfd45a873629109f24b29

Each transaction has:

- **Nonce** - int, used for prevent transaction reply
- **Gas Price** - big int, used for
- **Type** - type of transaction (see below)
- **Data** - data of transaction
- **Payload** (arbitrary bytes) - arbitrary user-defined bytes
- **Service Data** - service data
- **ECDSA fields (R, S and V)** - digital signature of transaction

.. code-block:: go

    type Transaction struct {
        Nonce       uint64
        GasPrice    *big.Int
        Type        byte
        Data        []byte
        Payload     []byte
        ServiceData []byte
        V           *big.Int
        R           *big.Int
        S           *big.Int
    }

Types
^^^^^

Type of transaction is determined by a single byte.

+----------------------------------+---------+
| Type Name                        | Byte    |
+==================================+=========+
| **TypeSend**                     | 0x01    |
+----------------------------------+---------+
| **TypeConvert**                  | 0x02    |
+----------------------------------+---------+
| **TypeCreateCoin**               | 0x03    |
+----------------------------------+---------+
| **TypeDeclareCandidacy**         | 0x04    |
+----------------------------------+---------+
| **TypeDelegate**                 | 0x05    |
+----------------------------------+---------+
| **TypeUnbond**                   | 0x06    |
+----------------------------------+---------+
| **TypeRedeemCheck**              | 0x07    |
+----------------------------------+---------+
| **TypeSetCandidateOnline**       | 0x08    |
+----------------------------------+---------+
| **TypeSetCandidateOffline**      | 0x09    |
+----------------------------------+---------+

Send transaction
^^^^^^^^^^^^^^^^

Transaction for sending arbitrary coin.

*Data field contents:*

.. code-block:: go

    type SendData struct {
        Coin  [10]byte
        To    [20]byte
        Value *big.Int
    }

Convert transaction
^^^^^^^^^^^^^^^^^^^

Transaction for converting one coin (owned by sender) to another coin in a system.

*Data field contents:*

.. code-block:: go

    type ConvertData struct {
        FromCoinSymbol [10]byte
        ToCoinSymbol   [10]byte
        Value          *big.Int
    }

Create coin transaction
^^^^^^^^^^^^^^^^^^^^^^^

Transaction for creating new coin in a system.

*Data field contents:*

.. code-block:: go

    type CreateCoinData struct {
        Name                 string
        Symbol               [10]byte
        InitialAmount        *big.Int
        InitialReserve       *big.Int
        ConstantReserveRatio uint
    }

Declare candidacy transaction
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Transaction for declaring new validator candidacy.

*Data field contents:*

.. code-block:: go

    type DeclareCandidacyData struct {
        Address    [20]byte
        PubKey     []byte
        Commission uint
        Coin       [10]byte
        Stake      *big.Int
    }

Delegate transaction
^^^^^^^^^^^^^^^^^^^^

Transaction for delegating funds to validator.

*Data field contents:*

.. code-block:: go

    type DelegateData struct {
        PubKey []byte
        Coin   [10]byte
        Stake  *big.Int
    }

Unbound transaction
^^^^^^^^^^^^^^^^^^^

Transaction for unbounding funds from validator's stake.

*Data field contents:*

.. code-block:: go

    type UnbondData struct {
        PubKey []byte
        Coin   [10]byte
        Value  *big.Int
    }

Redeem check transaction
^^^^^^^^^^^^^^^^^^^^^^^^

Transaction for redeeming a check.

*Data field contents:*

.. code-block:: go

    type RedeemCheckData struct {
        RawCheck []byte
        Proof    [65]byte
    }

Set candidate online transaction
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Transaction for turning candidate on.

*Data field contents:*

.. code-block:: go

    type SetCandidateOnData struct {
        PubKey []byte
    }

Set candidate offline transaction
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Transaction for turning candidate off.

*Data field contents:*

.. code-block:: go

    type SetCandidateOffData struct {
        PubKey []byte
    }

