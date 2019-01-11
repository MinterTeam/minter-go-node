Commissions
===========

For each transaction sender should pay fee. Fees are measured in "units".

1 unit = 10^15 pip = 0.001 bip.

Standard commissions
^^^^^^^^^^^^^^^^^^^^

Here is a list of current fees:

+----------------------------------+---------------------+
| Type                             | Fee                 |
+==================================+=====================+
| **TypeSend**                     | 10 units            |
+----------------------------------+---------------------+
| **TypeSellCoin**                 | 100 units           |
+----------------------------------+---------------------+
| **TypeSellAllCoin**              | 100 units           |
+----------------------------------+---------------------+
| **TypeBuyCoin**                  | 100 units           |
+----------------------------------+---------------------+
| **TypeCreateCoin**               | 1000 units          |
+----------------------------------+---------------------+
| **TypeDeclareCandidacy**         | 10000 units         |
+----------------------------------+---------------------+
| **TypeDelegate**                 | 100 units           |
+----------------------------------+---------------------+
| **TypeUnbond**                   | 100 units           |
+----------------------------------+---------------------+
| **TypeRedeemCheck**              | 30 units            |
+----------------------------------+---------------------+
| **TypeSetCandidateOnline**       | 100 units           |
+----------------------------------+---------------------+
| **TypeSetCandidateOffline**      | 100 units           |
+----------------------------------+---------------------+
| **TypeCreateMultisig**           | 100 units           |
+----------------------------------+---------------------+
| **TypeMultisend**                | 10+(n-1)*5 units    |
+----------------------------------+---------------------+
| **TypeEditCandidate**            | 10000 units         |
+----------------------------------+---------------------+

Also sender should pay extra 2 units per byte in Payload and Service Data fields.

Multisend transaction requires 1 unit per recipient excluding first one.

Special fees
^^^^^^^^^^^^

To issue a coin with short name Coiner should pay extra fee. Fee is depends on length of Coin Symbol.

| 3 letters – 1 000 000 bips + standard transaction fee
| 4 letters – 100 000 bips + standard transaction fee
| 5 letters – 10 000 bips + standard transaction fee
| 6 letters – 1000 bips + standard transaction fee
| 7 letters – 100 bips + standard transaction fee
| 8 letters – 10 bips + standard transaction fee
| 9-10 letters - just standard transaction fee
