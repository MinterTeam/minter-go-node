Validators
==========

Introduction
^^^^^^^^^^^^

...

Requirements
^^^^^^^^^^^^

...

Rules and fines
^^^^^^^^^^^^^^^

...

How to become validator in testnet
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

1. Install and run Minter Full Node. See :ref:`install-minter`. Make sure your node successfully synchronized.
2. Generate and install validator's key using our `tool <https://github.com/MinterTeam/minter-gen-validator>`__.
3. Restart Minter Node and Tendermint.
4. To to `Vault <http://vault.minter.network/>`__ (you can receive testnet coins in our telegram wallet @BipWallet_Bot) and send 2 transactions:

Declare candidacy
    - Address - You will receive rewards to this address and will be able to on/off your validator.
    - Public Key - Paste public key you created in step 2.
    - Commission - Set commission for delegated stakes.
    - Coin - Enter coin of your stake (MNT).
    - Stake - Enter value of your stake in given coin.

.. figure:: assets/vault-declare.png
    :width: 300px

Set candidate online
    Public Key - Paste public key you created in step 2.

.. figure:: assets/vault-candidate-on.png
    :width: 300px

5. Done.
    Now you will receive reward as long as your node is running and available.
