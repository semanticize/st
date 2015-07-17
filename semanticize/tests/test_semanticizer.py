import os
from nose.tools import assert_true
from tempfile import NamedTemporaryFile
from subprocess import call

from semanticize import Semanticizer

gopath = os.environ['GOPATH']
modelfile = NamedTemporaryFile()
wikidump = 'wikidump/nlwiki-20140927-sample.xml'
semmodel = modelfile.name
call([gopath + '/bin/semanticizest-dumpparser', semmodel, wikidump])


def test_semanticizer():
    sem = Semanticizer(model=semmodel, stPath=gopath + '/bin/semanticizest')
    sentence = 'Antwerpen'
    candidates = sem.all_candidates(sentence)
    assert_true(len(candidates) > 0, 'Should find some candidates')
