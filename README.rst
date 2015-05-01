|Travis|_

.. |Travis| image:: https://api.travis-ci.org/semanticize/st.png?branch=master
.. _Travis: https://travis-ci.org/semanticize/st


Semanticizer, standalone
========================

Semanticizest is a package for doing entity linking, also known as
semantic linking or semanticizing: you feed it text, and it outputs links
to pertinent Wikipedia concepts. You can use these links as a "semantic
representation" of the text for NLP or machine learning, or just to provide
some links to background info on the Wikipedia.

This is the Go version of semanticizest.


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

You now have a working parser at ``${GOPATH}/bin/dumpparser``. Issue::

    ${GOPATH}/bin/dumpparser --help

to figure out how to generate a semanticizer model, then use this model from
the REST API::

    ${GOPATH}/bin/semanticizest --http=:5002 your_model
    curl http://localhost:5002/all -d 'Does the entity linking work?'
