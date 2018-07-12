Commissions
===========

For each transaction sender should pay fee. Fees are measured in "units".

1 unit = 10^8 pip = 0.00000001 bip.

Standard commissions
^^^^^^^^^^^^^^^^^^^^

Here is a list of current fees:

+----------------------------------+---------------------+
| Type                             | Fee                 |
+==================================+=====================+
| **TypeSend**                     | 1 000 000 units     |
+----------------------------------+---------------------+
| **TypeSellCoin**                 | 10 000 000 units    |
+----------------------------------+---------------------+
| **TypeBuyCoin**                  | 10 000 000 units    |
+----------------------------------+---------------------+
| **TypeCreateCoin**               | 100 000 000 units   |
+----------------------------------+---------------------+
| **TypeDeclareCandidacy**         | 1 000 000 000 units |
+----------------------------------+---------------------+
| **TypeDelegate**                 | 10 000 000 units    |
+----------------------------------+---------------------+
| **TypeUnbond**                   | 10 000 000 units    |
+----------------------------------+---------------------+
| **TypeRedeemCheck**              | 1 000 000 units     |
+----------------------------------+---------------------+
| **TypeSetCandidateOnline**       | 10 000 000 units    |
+----------------------------------+---------------------+
| **TypeSetCandidateOffline**      | 10 000 000 units    |
+----------------------------------+---------------------+

Also sender should pay extra 200 000 units per byte in Payload and Service Data fields.

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
