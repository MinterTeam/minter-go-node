Transactions
============

Semantic
^^^^^^^^

Transactions in Minter are `RLP-encoded <https://github.com/ethereum/wiki/wiki/RLP>`__ structures.

Example of a signed transaction:

::

    f873230101aae98a4d4e540000000000000094a93163fdf10724dc4785ff5cbfb9
    ac0b5949409f880de0b6b3a764000080801ba06838db4a2197cfd70ede8d8d184d
    bf332932ca051a243eb7886791250e545dd3a04b63fb1d1b5ef5f2cbd2ea12530c
    da520b3280dcb75bfd45a873629109f24b29

Each transaction has:

- **Nonce** - int, used for prevent transaction reply.
- **Gas Price** - big int, used for managing transaction fees.
- **Gas Coin** - 10 bytes, symbol of a coin to pay fee
- **Type** - type of transaction (see below).
- **Data** - data of transaction (depends on transaction type).
- **Payload** (arbitrary bytes) - arbitrary user-defined bytes.
- **Service Data** - reserved field.
- **Signature Type** - single or multisig transaction.
- **Signature Data** - digital signature of transaction.

.. code-block:: go

    type Transaction struct {
        Nonce         uint64
        GasPrice      *big.Int
        GasCoin       [10]byte
        Type          byte
        Data          []byte
        Payload       []byte
        ServiceData   []byte
        SignatureType byte
        SignatureData Signature
    }

    type Signature struct {
        V           *big.Int
        R           *big.Int
        S           *big.Int
    }

    type MultiSignature struct {
        MultisigAddress [20]byte
        Signatures      []Signature
    }

Signature Types
^^^^^^^^^^^^^^^

+----------------------------------+---------+
| Type Name                        | Byte    |
+==================================+=========+
| **TypeSingle**                   | 0x01    |
+----------------------------------+---------+
| **TypeMulti**                    | 0x02    |
+----------------------------------+---------+

Types
^^^^^

Type of transaction is determined by a single byte.

+----------------------------------+---------+
| Type Name                        | Byte    |
+==================================+=========+
| **TypeSend**                     | 0x01    |
+----------------------------------+---------+
| **TypeSellCoin**                 | 0x02    |
+----------------------------------+---------+
| **TypeSellAllCoin**              | 0x03    |
+----------------------------------+---------+
| **TypeBuyCoin**                  | 0x04    |
+----------------------------------+---------+
| **TypeCreateCoin**               | 0x05    |
+----------------------------------+---------+
| **TypeDeclareCandidacy**         | 0x06    |
+----------------------------------+---------+
| **TypeDelegate**                 | 0x07    |
+----------------------------------+---------+
| **TypeUnbond**                   | 0x08    |
+----------------------------------+---------+
| **TypeRedeemCheck**              | 0x09    |
+----------------------------------+---------+
| **TypeSetCandidateOnline**       | 0x0A    |
+----------------------------------+---------+
| **TypeSetCandidateOffline**      | 0x0B    |
+----------------------------------+---------+
| **TypeCreateMultisig**           | 0x0C    |
+----------------------------------+---------+
| **TypeMultisend**                | 0x0D    |
+----------------------------------+---------+
| **TypeEditCandidate**            | 0x0E    |
+----------------------------------+---------+

Send transaction
^^^^^^^^^^^^^^^^

Type: **0x01**

Transaction for sending arbitrary coin.

*Data field contents:*

.. code-block:: go

    type SendData struct {
        Coin  [10]byte
        To    [20]byte
        Value *big.Int
    }

| **Coin** - Symbol of a coin.
| **To** - Recipient address in Minter Network.
| **Value** - Amount of **Coin** to send.

Sell coin transaction
^^^^^^^^^^^^^^^^^^^^^

Type: **0x02**

Transaction for selling one coin (owned by sender) in favour of another coin in a system.

*Data field contents:*

.. code-block:: go

    type SellCoinData struct {
        CoinToSell          [10]byte
        ValueToSell         *big.Int
        CoinToBuy           [10]byte
        MinimumValueToBuy   *big.Int
    }

| **CoinToSell** - Symbol of a coin to give.
| **ValueToSell** - Amount of **CoinToSell** to give.
| **CoinToBuy** - Symbol of a coin to get.
| **MinimumValueToBuy** - Minimum value of coins to get.

Sell all coin transaction
^^^^^^^^^^^^^^^^^^^^^^^^^

Type: **0x03**

Transaction for selling all existing coins of one type (owned by sender) in favour of another coin in a system.

*Data field contents:*

.. code-block:: go

    type SellAllCoinData struct {
        CoinToSell          [10]byte
        CoinToBuy           [10]byte
        MinimumValueToBuy   *big.Int
    }

| **CoinToSell** - Symbol of a coin to give.
| **CoinToBuy** - Symbol of a coin to get.
| **MinimumValueToBuy** - Minimum value of coins to get.

Buy coin transaction
^^^^^^^^^^^^^^^^^^^^

Type: **0x04**

Transaction for buy a coin paying another coin (owned by sender).

*Data field contents:*

.. code-block:: go

    type BuyCoinData struct {
        CoinToBuy           [10]byte
        ValueToBuy          *big.Int
        CoinToSell          [10]byte
        MaximumValueToSell  *big.Int
    }

| **CoinToBuy** - Symbol of a coin to get.
| **ValueToBuy** - Amount of **CoinToBuy** to get.
| **CoinToSell** - Symbol of a coin to give.
| **MaximumValueToSell** - Maximum value of coins to sell.

Create coin transaction
^^^^^^^^^^^^^^^^^^^^^^^

Type: **0x05**

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

| **Name** - Name of a coin. Arbitrary string up to 64 letters length.
| **Symbol** - Symbol of a coin. Must be unique, alphabetic, uppercase, 3 to 10 symbols length.
| **InitialAmount** - Amount of coins to issue. Issued coins will be available to sender account.
| **InitialReserve** - Initial reserve in BIP's.
| **ConstantReserveRatio** - CRR, uint, should be from 10 to 100.

Declare candidacy transaction
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Type: **0x06**

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

| **Address** - Address of candidate in Minter Network. This address would be able to control candidate. Also all rewards will be sent to this address.
| **PubKey** - Public key of a validator.
| **Commission** - Commission (from 0 to 100) from rewards which delegators will pay to validator.
| **Coin** - Symbol of coin to stake.
| **Stake** - Amount of coins to stake.

Delegate transaction
^^^^^^^^^^^^^^^^^^^^

Type: **0x07**

Transaction for delegating funds to validator.

*Data field contents:*

.. code-block:: go

    type DelegateData struct {
        PubKey []byte
        Coin   [10]byte
        Stake  *big.Int
    }

| **PubKey** - Public key of a validator.
| **Coin** - Symbol of coin to stake.
| **Stake** - Amount of coins to stake.

Unbond transaction
^^^^^^^^^^^^^^^^^^

Type: **0x08**

Transaction for unbonding funds from validator's stake.

*Data field contents:*

.. code-block:: go

    type UnbondData struct {
        PubKey []byte
        Coin   [10]byte
        Value  *big.Int
    }

| **PubKey** - Public key of a validator.
| **Coin** - Symbol of coin to unbond.
| **Value** - Amount of coins to unbond.

Redeem check transaction
^^^^^^^^^^^^^^^^^^^^^^^^

Type: **0x09**

Transaction for redeeming a check.

*Data field contents:*

.. code-block:: go

    type RedeemCheckData struct {
        RawCheck []byte
        Proof    [65]byte
    }

| **RawCheck** - Raw check received from sender.
| **Proof** - Proof of owning a check.

Set candidate online transaction
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Type: **0x0A**

Transaction for turning candidate on. This transaction should be sent from address which is set in the "Declare candidacy transaction".

*Data field contents:*

.. code-block:: go

    type SetCandidateOnData struct {
        PubKey []byte
    }

| **PubKey** - Public key of a validator.

Set candidate offline transaction
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Type: **0x0B**

Transaction for turning candidate off. This transaction should be sent from address which is set in the "Declare candidacy transaction".

*Data field contents:*

.. code-block:: go

    type SetCandidateOffData struct {
        PubKey []byte
    }

| **PubKey** - Public key of a validator.

Create multisig address
^^^^^^^^^^^^^^^^^^^^^^^

Type: **0x0C**

Transaction for creating multisignature address.

*Data field contents:*

.. code-block:: go

    type CreateMultisigData struct {
        Threshold uint
        Weights   []uint
        Addresses [][20]byte
    }


Multisend transaction
^^^^^^^^^^^^^^^^^^^^^

Type: **0x0D**

Transaction for sending coins to multiple addresses.

*Data field contents:*

.. code-block:: go

    type MultisendData struct {
        List []MultisendDataItem
    }

    type MultisendDataItem struct {
        Coin  [10]byte
        To    [20]byte
        Value *big.Int
    }

Edit candidate transaction
^^^^^^^^^^^^^^^^^^^^^^^^^^

Type: **0x0E**

Transaction for editing existing candidate

*Data field contents:*

.. code-block:: go

    type EditCandidateData struct {
        PubKey           []byte
        RewardAddress    [20]byte
        OwnerAddress     [20]byte
    }
