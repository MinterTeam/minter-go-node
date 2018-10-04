Multisignatures
===============

Minter has built-in support for multisignature wallets. Multisignatures, or technically
Accountable Subgroup Multisignatures (ASM), are signature schemes which enable any
subgroup of a set of signers to sign any message, and reveal to the verifier exactly
who the signers were.

Suppose the set of signers is of size *n*. If we validate a signature if any subgroup
of size *k* signs a message, this becomes what is commonly reffered to as a *k* of *n*
multisig in Bitcoin.

Minter Multisig Wallets has 2 main goals:
    - Atomic swaps with sidechains
    - Basic usage to manage funds within Minter Blockchain

Structure of multisig wallet
^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Each multisig wallet has:
    - Set of signers with corresponding weights
    - Threshold
Transactions from multisig wallets are proceed identically to the K of N multisig in Bitcoin,
except the multisig fails if the sum of the weights of signatures is less than the threshold.


How to create multisig wallet
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

TO BE DESCRIBED

How to use multisig wallet
^^^^^^^^^^^^^^^^^^^^^^^^^^

TO BE DESCRIBED
