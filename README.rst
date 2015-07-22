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

    go get -u github.com/semanticize/st/...

You now have a working parser at ``${GOPATH}/bin/semanticizest-dumpparser``.
Issue::

    ${GOPATH}/bin/semanticizest-dumpparser --help

to figure out how to generate a semanticizer model, then use this model from
the REST API::

    ${GOPATH}/bin/semanticizest --http=:5002 your_model
    curl http://localhost:5002/all -d 'Does the entity linking work?'

You can also use semanticizest as a command-line tool by omitting ``--http``.
In that case, it will read paragraphs (double newline-separated) from standard
input and emit a JSON representation of the candidate entities in each
paragraph.

Python binding
==============

Once you have built a semanticizer model, you can use semanticizest from Python
using the Python wrapper. Install the Python wrapper with::

    python setup.py install

You can use the python wrapper as follows:

The wrapper can operate in two modes: either you start the semanticizest
REST server, or you have the wrapper start it for you.

If you already have a server running on port 5002, as described above::

    from semanticizest import Client

    sentence = 'Antwerpen'
    url = 'http://localhost:5002/'
    client = Client(url)
    candidates = client.all_candidates(sentence)

To start the server automatically, you need to construct a ``Semanticizer``
from the path to the model file::

    from semanticizest import Semanticizer

    sentence = 'Antwerpen'
    semanticizer = Semanticizer('/path/to/model.db')
    candidates = client.all_candidates(sentence)
