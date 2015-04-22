|Travis|_

.. |Travis| image:: https://api.travis-ci.org/semanticize/st.png?branch=master
.. _Travis: https://travis-ci.org/semanticize/st


Wikipedia dump parser for semanticizest
=======================================

This program parses Wikipedia database dumps for consumption by semanticizest.


Installing
----------

Make sure you have a Go compiler (1.2 or newer) and Git.
On Debian/Ubuntu/Mint, that's::

    sudo apt-get install git golang-go

On CentOS::

    sudo yum -y install git golang

Set up a Go workspace, if you haven't already. For example::

    mkdir /some/where/go
    cd /some/where/go
    export GOPATH=$(pwd)

Fetch and compile::

    go get github.com/semanticize/st
    go install github.com/semanticize/st/dumpparser
    go install github.com/semanticize/st/semanticizest

You now have a working parser at ``${GOPATH}/bin/dumpparser``.
