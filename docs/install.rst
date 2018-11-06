.. _install-minter:

Install Minter
==============

There are several ways you can install Minter Blockchain Testnet node on your machine:

Using binary
------------

Download Minter
^^^^^^^^^^^^^^^

Get `latest binary build <https://github.com/MinterTeam/minter-go-node/releases>`__ suitable for your architecture and
unpack it to desired folder.

Run Minter
^^^^^^^^^^

.. code-block:: bash
    :lineno-start: 13

    ./minter

Then open http://localhost:3000/ in local browser to see node's GUI.

Using Docker
------------

You'll need `docker <https://docker.com/>`__ and `docker compose <https://docs.docker.com/compose/>`__ installed.

Clone Minter source code to your machine
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

.. code-block:: bash
    :lineno-start: 1

    git clone https://github.com/MinterTeam/minter-go-node.git
    cd minter-go-node

Start Minter
^^^^^^^^^^^^

.. code-block:: bash
    :lineno-start: 10

    docker-compose up

Then open http://localhost:3000/ in local browser to see node's GUI.

From Source
-----------

You'll need ``go`` `installed <https://golang.org/doc/install>`__ and the required
`environment variables set <https://github.com/tendermint/tendermint/wiki/Setting-GOPATH>`__

Clone Minter source code to your machine
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

.. code-block:: bash
    :lineno-start: 1

    mkdir -p $GOPATH/src/github.com/MinterTeam
    cd $GOPATH/src/github.com/MinterTeam
    git clone https://github.com/MinterTeam/minter-go-node.git
    cd minter-go-node

Get Tools & Dependencies
^^^^^^^^^^^^^^^^^^^^^^^^

.. code-block:: bash
    :lineno-start: 5

    make get_tools
    make get_vendor_deps

Compile
^^^^^^^

.. code-block:: bash
    :lineno-start: 7

    make install

to put the binary in ``$GOPATH/bin`` or use:

.. code-block:: bash
    :lineno-start: 8

    make build

to put the binary in ``./build``.

The latest ``minter version`` is now installed.

Run Minter
^^^^^^^^^^

.. code-block:: bash
    :lineno-start: 13

    minter

Then open http://localhost:3000/ in local browser to see node's GUI.
