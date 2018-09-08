Delegator FAQ
=============

What is a delegator?
^^^^^^^^^^^^^^^^^^^^

People that cannot, or do not want to run validator operations, can still participate in
the staking process as delegators. Indeed, validators are not chosen based on their own
stake but based on their total stake, which is the sum of their own stake and of the stake
that is delegated to them. This is an important property, as it makes delegators a
safeguard against validators that exhibit bad behavior. If a validator misbehaves, its
delegators will move their staked coins away from it, thereby reducing its stake. Eventually,
if a validator's stake falls under the top addresses with highest stake, it will exit the
validator set.

Delegators share the revenue of their validators, but they also share the risks. In terms
of revenue, validators and delegators differ in that validators can apply a commission on
the revenue that goes to their delegator before it is distributed. This commission is
known to delegators beforehand and cannot be changed. In terms of risk, delegators' coins
can be slashed if their validator misbehaves. For more, see Risks section.

To become delegators, coin holders need to send a "Delegate transaction" where they specify
how many coins they want to bond and to which validator. Later, if a delegator wants to
unbond part or all of its stake, it needs to send an "Unbond transaction". From there, the
delegator will have to wait 30 days to retrieve its coins.

Directives of delegators
^^^^^^^^^^^^^^^^^^^^^^^^

Being a delegator is not a passive task. Here are the main directives of a delegator:

- Perform careful due diligence on validators before delegating. If a validator misbehaves,
  part of its total stake, which includes the stake of its delegators, can be slashed. Delegators
  should therefore carefully select validators they think will behave correctly.

- Actively monitor their validator after having delegated. Delegators should ensure that the
  validators they're delegating to behaves correctly, meaning that they have good uptime, do not
  get hacked and participate in governance. If a delegator is not satisfied with its validator,
  it can unbond and switch to another validator.

Revenue
^^^^^^^

Validators and delegators earn revenue in exchange for their services. This revenue is given in three forms:

- Block rewards
- Transaction fees: Each transaction on the Minter Network comes with transactions fees. Fees are distributed to
  validators and delegators in proportion to their stake.

Validator's commission
^^^^^^^^^^^^^^^^^^^^^^

Each validator's staking pool receives revenue in proportion to its total stake. However, before this revenue is
distributed to delegators inside the staking pool, the validator can apply a commission. In other words, delegators
have to pay a commission to their validators on the revenue they earn.

``10%`` from reward going to DAO account.

``10%`` from reward going to Developers.

Lets consider a validator whose stake (i.e. self-bonded stake + delegated stake) is 10% of the total stake of all
validators. This validator has 20% self-bonded stake and applies a commission of 10%. Now let us consider a block
with the following revenue:

- 111 Bips as block reward (after subtraction taxes of 20%)
- 10 Bips as transaction fees (after subtraction taxes of 20%)

This amounts to a total of 121 Bips to be distributed among all staking pools.

Our validator's staking pool represents 10% of the total stake, which means the pool obtains 12.1 bips. Now let us
look at the internal distribution of revenue:

- Commission = 10% * 80% * 12.1 bips = 0.69696 bips
- Validator's revenue = 20% * 12.1 bips + Commission = 3.11696 bips
- Delegators' total revenue = 80% * 12.1 bips - Commission = 8.98304 bips

Then, each delegator in the staking pool can claim its portion of the delegators' total revenue.

Risks
^^^^^

Staking coins is not free of risk. First, staked coins are locked up, and retrieving them requires a 30 days waiting
period called unbonding period. Additionally, if a validator misbehaves, a portion of its total stake can be slashed
(i.e. destroyed). This includes the stake of their delegators.

There are 2 main slashing conditions:

- **Double signing**: If someone reports on chain A that a validator signed two blocks at the same height on chain
  A and chain B, this validator will get slashed on chain A
- **Unavailability**: If a validator's signature has not been included in the last 12 blocks,
  1% of stake will get slashed and validator will be turned off

This is why delegators should perform careful due diligence on validators before delegating. It is also important
that delegators actively monitor the activity of their validators. If a validator behaves suspiciously or is too
often offline, delegators can choose to unbond from it or switch to another validator. Delegators can also mitigate
risk by distributing their stake across multiple validators.
