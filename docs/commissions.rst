Commissions
===========

For each transaction sender should pay fee. Fees are measured in "units".

1 unit = 10^8 pip = 0.00000001 bip.

Standard commissions
^^^^^^^^^^^^^^^^^^^^

Here is a list of current fees:

+----------------------------------+--------------+
| Type                             | Fee          |
+==================================+==============+
| **TypeSend**                     | 1000 units   |
+----------------------------------+--------------+
| **TypeConvert**                  | 10000 units  |
+----------------------------------+--------------+
| **TypeCreateCoin**               | 100000 units |
+----------------------------------+--------------+
| **TypeDeclareCandidacy**         | 100000 units |
+----------------------------------+--------------+
| **TypeDelegate**                 | 10000 units  |
+----------------------------------+--------------+
| **TypeUnbond**                   | 10000 units  |
+----------------------------------+--------------+
| **TypeRedeemCheck**              | 1000 units   |
+----------------------------------+--------------+
| **TypeSetCandidateOnline**       | 500 units    |
+----------------------------------+--------------+
| **TypeSetCandidateOffline**      | 1000 units   |
+----------------------------------+--------------+

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
