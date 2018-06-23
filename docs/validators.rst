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

1. Install and run Minter Full Node.
    See :ref:`install-minter`. Make sure your node successfully synchronized.

2. Generate and install validator's key using our `tool <https://github.com/MinterTeam/minter-gen-validator>`__.
    If you already have ``priv_validator.json`` file – just replace it with new one.

3. Restart Minter Node and Tendermint.
    Restarting will apply changes to ``priv_validator.json`` file.

4. Go to `Vault <http://vault.minter.network/>`__ and send 2 transactions:
    Fill and send ``Declare candidacy`` and ``Set candidate online`` forms.

    If you cannot open Vault because of invalid certificate:
    `reset HSTS <https://www.thesslstore.com/blog/clear-hsts-settings-chrome-firefox/>`__ for domains
    ``minter.network`` and ``vault.minter.network``. Then try to open
    `HTTP version of Vault <http://vault.minter.network/>`__.

    P.S. You can receive testnet coins in our telegram wallet @BipWallet_Bot.

    1. Declare candidacy
        - Address - You will receive rewards to this address and will be able to on/off your validator.
        - Public Key - Paste public key you created in step 2.
        - Commission - Set commission for delegated stakes.
        - Coin - Enter coin of your stake (MNT).
        - Stake - Enter value of your stake in given coin.

    .. figure:: assets/vault-declare.png
        :width: 300px

    2. Set candidate online
        Public Key - Paste public key you created in step 2.

    .. figure:: assets/vault-candidate-on.png
        :width: 300px

5. Done.
    Now you will receive reward as long as your node is running and available.
