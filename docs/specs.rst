Blockchain Specification
========================

Tendermint
^^^^^^^^^^

Minter Blockchain utilizes ``Tendermint Consensus Engine``.

Tendermint is software for securely and consistently replicating an application on many machines.
By securely, we mean that Tendermint works even if up to 1/3 of machines fail in arbitrary ways.
By consistently, we mean that every non-faulty machine sees the same transaction log and computes the same state.
Secure and consistent replication is a fundamental problem in distributed systems; it plays a critical role in the
fault tolerance of a broad range of applications, from currencies, to elections, to infrastructure orchestration,
and beyond.

Tendermint is designed to be easy-to-use, simple-to-understand, highly performant, and useful for a wide variety of
distributed applications.

You can read more about Tendermint Consensus in `official documentation <https://tendermint.com/docs/>`__.

Consensus
^^^^^^^^^

In Minter we implemented Delegated Proof of Stake (DPOS) Consensus Protocol.

DPOS is the fastest, most efficient, most decentralized, and most flexible consensus model available. DPOS leverages
the power of stakeholder approval voting to resolve consensus issues in a fair and democratic way.

Block speed
^^^^^^^^^^^

Minter Blockchain is configured to produce ``1 block per 5 sec``. Actual block speed may vary depends on validators count,
their computational power, internet speed, etc.

Block size
^^^^^^^^^^

We limit block size to ``10 000 transactions``. Block size in terms of bytes is not limited.

