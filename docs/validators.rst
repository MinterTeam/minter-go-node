Validators
==========

Introduction
^^^^^^^^^^^^

The Minter Blockchain is based on Tendermint, which relies on a set of validators that are
responsible for committing new blocks in the blockchain. These validators participate in
the consensus protocol by broadcasting votes which contain cryptographic signatures signed
by each validator’s private key.

Validator candidates can bond their own coins and have coins “delegated”, or staked, to them
by token holders. The validators are determined by who has the most stake delegated to them.

Validators and their delegators will earn BIP (MNT) as rewards for blocks and commissions. Note
that validators can set commission on the rewards their delegators receive as additional incentive.

If validators double sign, are frequently offline or do not participate in governance, their
staked coins (including coins of users that delegated to them) can be slashed. The penalty
depends on the severity of the violation.

Requirements
^^^^^^^^^^^^

Minimal requirements for running Validator's Node are:

- 2GB RAM
- 100GB of disk space
- 1.4 GHz 2v CPU

SSD disks are preferable for high transaction throughput.

Recommended:

- 4GB RAM
- 200GB SSD
- x64 2.0 GHz 4v CPU


Rewards
^^^^^^^

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

    4.1. Declare candidacy
        - Address - You will receive rewards to this address and will be able to on/off your validator.
        - Public Key - Paste public key you created in step 2.
        - Commission - Set commission for delegated stakes.
        - Coin - Enter coin of your stake (MNT).
        - Stake - Enter value of your stake in given coin.

    .. figure:: assets/vault-declare.png
        :width: 300px

    4.2. Set candidate online
        Public Key - Paste public key you created in step 4.2.

        *Note: You should send transaction from address you choose in Address field in step 4.2*

    .. figure:: assets/vault-candidate-on.png
        :width: 300px

5. Done.
    Now you will receive reward as long as your node is running and available.
