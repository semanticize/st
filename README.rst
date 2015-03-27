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

Fetch and compile the dump parser::

    go get github.com/semanticize/dumpparser/dumpparser
    go install github.com/semanticize/dumpparser/dumpparser

You now have a working parser at ``${GOPATH}/bin/dumpparser``.
