Running in production
=====================

DOS Exposure and Mitigation
---------------------------

Validators are supposed to setup `Sentry Node Architecture
<https://blog.cosmos.network/tendermint-explained-bringing-bft-based-pos-to-the-public-blockchain-domain-f22e274a0fdb>`__
to prevent Denial-of-service attacks. You can read more about it `here
<https://github.com/tendermint/aib-data/blob/develop/medium/TendermintBFT.md>`__.

P2P
~~~

The core of the Tendermint peer-to-peer system is ``MConnection``. Each
connection has ``MaxPacketMsgPayloadSize``, which is the maximum packet size
and bounded send & receive queues. One can impose restrictions on send &
receive rate per connection (``SendRate``, ``RecvRate``).

RPC
~~~

Endpoints returning multiple entries are limited by default to return 30
elements (100 max).

Rate-limiting and authentication are another key aspects to help protect
against DOS attacks. While in the future we may implement these features, for
now, validators are supposed to use external tools like `NGINX
<https://www.nginx.com/blog/rate-limiting-nginx/>`__ or `traefik
<https://docs.traefik.io/configuration/commons/#rate-limiting>`__ to achieve
the same things.

Monitoring Tendermint
---------------------

Each Tendermint instance has a standard `/health` RPC endpoint, which responds
with 200 (OK) if everything is fine and 500 (or no response) - if something is
wrong.

Other useful endpoints include mentioned earlier `/status`, `/net_info` and
`/validators`.

We have a small tool, called tm-monitor, which outputs information from the
endpoints above plus some statistics. The tool can be found `here
<https://github.com/tendermint/tools/tree/master/tm-monitor>`__.


Monitoring Minter
-----------------

Each Minter instance has a standard `/status` RPC endpoint, which responds
with 200 (OK) if everything is fine and 500 (or no response) - if something is
wrong.

What happens when my app dies?
------------------------------

You are supposed to run Tendermint and Minter under a `process supervisor
<https://en.wikipedia.org/wiki/Process_supervision>`__ (like systemd or runit).
It will ensure Tendermint and Minter is always running (despite possible errors).

Signal handling
---------------

We catch SIGINT and SIGTERM and try to clean up nicely. For other signals we
use the default behaviour in Go: `Default behavior of signals in Go programs
<https://golang.org/pkg/os/signal/#hdr-Default_behavior_of_signals_in_Go_programs>`__.

Hardware
--------

Processor and Memory
~~~~~~~~~~~~~~~~~~~~

Minimal requirements are:

- 2GB RAM
- 100GB of disk space
- 1.4 GHz CPU

SSD disks are preferable for high transaction throughput.

Recommended:

- 4GB RAM
- 200GB SSD
- x64 2.0 GHz 4v CPU

Operating Systems
~~~~~~~~~~~~~~~~~

Tendermint and Minter can be compiled for a wide range of operating systems thanks to Go
language (the list of $OS/$ARCH pairs can be found `here
<https://golang.org/doc/install/source#environment>`__).

While we do not favor any operation system, more secure and stable Linux server
distributions (like Centos) should be preferred over desktop operation systems
(like Mac OS).

Configuration parameters
------------------------

...
