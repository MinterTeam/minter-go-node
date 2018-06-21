Coins
=====

Minter Blockchain is multi-coin system.

Base coin in testnet is ``MNT``.
Base coin in mainnet is ``BIP``.

Coin Issuance
^^^^^^^^^^^^^

Every user of Minter can issue own coin. Each coin is backed by base coin in some proportion.
Issue own coin is as simple as filling a form with given fields:

- Coin name
- Coin symbol
- Initial supply
- Initial reserve
- Constant Reserve Ratio (CRR)

After coin issued you can send is as ordinary coin using standard wallets.

Coin Exchange
^^^^^^^^^^^^^

Each coin in system can be instantly exchanged to another coin. This is possible because each coin has "reserve" in base
coin.

Here are some formulas we are using for coin conversion:

CalculatePurchaseReturn
    Given a coin supply (s), reserve balance (r), CRR (c) and a deposit amount (d),
    calculates the return for a given conversion (in the base coin):

::

    return s * ((1 + d / r) ^ d - 1);


CalculateSaleReturn
    Given a coin supply (s), reserve balance (r), CRR (c) and a sell amount (a),
    calculates the return for a given conversion

::

    return r * (1 - (1 - a / s) ^ (1 / c));