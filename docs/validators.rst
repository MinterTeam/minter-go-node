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

If validators double sign or frequently offline, their staked coins (including coins of users that
delegated to them) can be slashed. The penalty depends on the severity of the violation.

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

Validators limitations
^^^^^^^^^^^^^^^^^^^^^^

Minter Network has limited number of available slots for validators.

At genesis there will be just ``16`` of them. ``4`` slots will be added each ``518,400`` blocks.
Maximum validators count is ``256``.

Rewards
^^^^^^^

Rewards for blocks and commissions are accumulated and proportionally (based on stake value)
payed once per ``12 blocks`` (approx 1 minute) to all active validators (and their delegators).

Block rewards are configured to decrease from 111 to 0 BIP (MNT) in 7 years.

Delegators receive their rewards at the same time after paying commission to their validators
(commission value is based on validator's settings).

``5%`` from reward going to DAO account.

Rules and fines
^^^^^^^^^^^^^^^

Validators have one main responsibility:

- Be able to constantly run a correct version of the software: validators need to make sure that their
  servers are always online and their private keys are not compromised.


If a validator misbehaves, its bonded stake along with its delegators' stake and will be slashed.
The severity of the punishment depends on the type of fault. There are 3 main faults that can result in slashing
of funds for a validator and its delegators:

- **Double signing**: If someone reports on chain A that a validator signed two blocks at the same height on chain
  A and chain B, this validator will get slashed on chain A
- **Unavailability**: If a validator's signature has not been included in the last X blocks,
  1% of stake will get slashed and validator will be turned off

Note that even if a validator does not intentionally misbehave, it can still be slashed if its node crashes,
looses connectivity, gets DDOSed, or if its private key is compromised.

Becoming validator in testnet
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

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
        Validators should declare their candidacy, after which users can delegate
        and, if they so wish, unbond. Then declaring candidacy validator should fill a form:

        - Address - You will receive rewards to this address and will be able to on/off your validator.
        - Public Key - Paste public key you created in step 2 *(Mp...)*.
        - Commission - Set commission for delegated stakes.
        - Coin - Enter coin of your stake (MNT).
        - Stake - Enter value of your stake in given coin.

    .. figure:: assets/vault-declare.png
        :width: 300px

    4.2. Set candidate online
        Validator is **offline** by default. When offline, validator is not included in the list of
        Minter Blockchain validators, so he is not receiving any rewards and cannot be punished
        for low availability.

        To turn your validator **on**, you should provide Public Key (which you created in step
        2 *(Mp...)*).

        *Note: You should send transaction from address you choose in Address field in step 4.2*

    .. figure:: assets/vault-candidate-on.png
        :width: 300px

5. Done.
    Now you will receive reward as long as your node is running and available.


DDOS protection. Sentry node architecture
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Denial-of-service attacks occur when an attacker sends a flood of internet traffic to an IP
address to prevent the server at the IP address from connecting to the internet.

An attacker scans the network, tries to learn the IP address of various validator
nodes and disconnect them from communication by flooding them with traffic.

One recommended way to mitigate these risks is for validators to carefully
structure their network topology in a so-called sentry node architecture.

Validator nodes should only connect to full-nodes they trust because they
operate them themselves or are run by other validators they know socially.
A validator node will typically run in a data center. Most data centers provide
direct links the networks of major cloud providers. The validator can use
those links to connect to sentry nodes in the cloud. This shifts the burden
of denial-of-service from the validator's node directly to its sentry nodes,
and may require new sentry nodes be spun up or activated to mitigate attacks
on existing ones.

Sentry nodes can be quickly spun up or change their IP addresses. Because
the links to the sentry nodes are in private IP space, an internet based
attacked cannot disturb them directly. This will ensure validator block
proposals and votes always make it to the rest of the network.

It is expected that good operating procedures on that part of validators will
completely mitigate these threats.

Practical instructions
----------------------

To setup your sentry node architecture you can follow the instructions below:

Validators nodes should edit their ``config.toml``:

::

        # Comma separated list of nodes to keep persistent connections to
        # Do not add private peers to this list if you don't want them advertised
        persistent_peers =[list of sentry nodes]

        # Set true to enable the peer-exchange reactor
        pex = false

Sentry Nodes should edit their ``config.toml``:

::

        # Comma separated list of peer IDs to keep private (will not be gossiped to other peers)
        private_peer_ids = "ipaddress of validator nodes"
