import os
import os.path
import re
import subprocess
from tempfile import NamedTemporaryFile

from nose.tools import assert_true

from semanticize import Semanticizer, SemanticizerServer


def gobin(name):
    return os.path.join(os.environ['GOPATH'], 'bin', name)


# XXX This leaves junk behind, should clean up.
modelfile = NamedTemporaryFile(prefix='semanticizer-test-')
wikidump = '../wikidump/nlwiki-20140927-sample.xml'
model = modelfile.name
if subprocess.call([gobin('semanticizest-dumpparser'), model, wikidump]) != 0:
    raise RuntimeError('dumpparser failed')


def test_semanticizer_server():
    server = SemanticizerServer(model=model,
                                serverpath=gobin('semanticizest'))
    server.terminate()
    assert_true(re.match(r'http://(localhost|127\.0\.0\.1):[0-9]+', server.url))


def test_semanticizer():
    with SemanticizerServer(model=model,
                            serverpath=gobin('semanticizest')) as server:
        client = Semanticizer(server)
        sentence = 'Antwerpen'
        candidates = client.all_candidates(sentence)
    assert_true(len(candidates) > 0, 'Should find some candidates')
