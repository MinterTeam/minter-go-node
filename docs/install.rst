Install Minter
==================

There are several ways you can install Minter Blockchain node on your machine:

Using Docker
----------------

You'll need `docker <https://docker.com/>`__ and `docker compose <https://docs.docker.com/compose/>`__ installed.

Clone Minter source code to your machine
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

::

    git clone https://github.com/MinterTeam/minter-go-node.git
    cd minter-go-node


Prepare folders and configs
^^^^^^^^^^^^^^^^^^^^^^^^^^^

::

    mkdir -p ~/.tendermint/data
    mkdir -p ~/.minter/data

    cp -R networks/testnet/ ~/.tendermint/config

    chmod -R 0777 ~/.tendermint
    chmod -R 0777 ~/.minter

Start Minter
^^^^^^^^^^^^

::

    docker-compose up


From Source
-----------

You'll need ``go`` `installed <https://golang.org/doc/install>`__ and the required
`environment variables set <https://github.com/tendermint/tendermint/wiki/Setting-GOPATH>`__

Install Tendermint 0.20
^^^^^^^^^^^^^^^^^^^^^^^
`Read official instructions <https://tendermint.readthedocs.io/en/master/install.html>`__

Clone Minter source code to your machine
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

::

    mkdir $GOPATH/src/github.com/MinterTeam
    cd $GOPATH/src/github.com/MinterTeam
    git clone https://github.com/MinterTeam/minter-go-node.git
    cd minter-go-node

Get Tools & Dependencies
^^^^^^^^^^^^^^^^^^^^^^^^

::

    make get_tools
    make get_vendor_deps

Compile
^^^^^^^

::

    make install

to put the binary in ``$GOPATH/bin`` or use:

::

    make build

to put the binary in ``./build``.

The latest ``minter version`` is now installed.

Create data directories
^^^^^^^^^^^^^^^^^^^^^^^

::

    mkdir -p ~/.tendermint/data
    mkdir -p ~/.minter/data

Copy config and genesis file
^^^^^^^^^^^^^^^^^^^^^^^^^^^^

::

    cp -R networks/testnet/ ~/.tendermint/config

Run Tendermint
^^^^^^^^^^^^^^

::

    tendermint node

Run Minter
^^^^^^^^^^^^^^

::

    minter
