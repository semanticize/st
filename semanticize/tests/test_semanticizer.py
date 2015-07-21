import os
from nose.tools import assert_true
from tempfile import NamedTemporaryFile
from subprocess import call

from semanticize import SemanticizerServer, SemanticizerClient

gopath = os.environ['GOPATH']
modelfile = NamedTemporaryFile()
wikidump = 'wikidump/nlwiki-20140927-sample.xml'
semmodel = modelfile.name
call([gopath + '/bin/semanticizest-dumpparser', semmodel, wikidump])


def test_semanticizerServer():
    server = SemanticizerServer(model='nlsample.go.model',
                                stPath='./bin/semanticizest')
    url = server.getURL()
    server.stop()
    assert_true(len(url) > 0, 'Should have a URL')


def test_semanticizer():
    server = SemanticizerServer(model='nlsample.go.model',
                                stPath='./bin/semanticizest')
    client = SemanticizerClient(server.getURL())
    sentence = 'Antwerpen'
    candidates = client.all_candidates(sentence)
    server.stop()
    assert_true(len(candidates) > 0, 'Should find some candidates')
